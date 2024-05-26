package kubeproxy

import (
	"bufio"
	"context"
	"fmt"
	"github.com/vishvananda/netlink"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/runtime"
	"minikubernetes/pkg/kubeproxy/route"
	"minikubernetes/pkg/kubeproxy/types"
	"minikubernetes/pkg/utils"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Proxy struct {
	ipvs         route.IPVS
	hostIP       string
	nameserverIP string
	// 缓存service full name到ClusterIP的映射
	serviceCache map[string]string
}

func NewKubeProxy() (*Proxy, error) {
	p := &Proxy{}
	ipvs, err := route.NewIPVS()
	if err != nil {
		return nil, err
	}
	p.ipvs = ipvs
	p.serviceCache = make(map[string]string)
	return p, nil
}

func (p *Proxy) Run(ctx context.Context, wg *sync.WaitGroup,
	serviceUpdates <-chan *types.ServiceUpdate, dnsUpdates <-chan *types.DNSUpdate) {
	log.Printf("KubeProxy running...")
	err := p.ipvs.Init()
	if err != nil {
		log.Printf("Failed to init ipvs: %v", err)
		return
	}
	log.Printf("IPVS initialized.")
	// get host ip
	hostIP, err := getHostIP()
	if err != nil {
		log.Printf("Failed to get host ip: %v", err)
		return
	}
	p.hostIP = hostIP
	nameserverIP, err := runtime.GetContainerBridgeIP("coredns")
	if err != nil {
		log.Printf("Failed to get nameserver ip: %v", err)
		return
	}
	p.nameserverIP = nameserverIP
	// 写入/etc/resolv.conf
	err = writeResolvConf(nameserverIP)
	if err != nil {
		log.Printf("Failed to write resolv.conf: %v", err)
		return
	}
	p.syncLoop(ctx, wg, serviceUpdates, dnsUpdates)
}

func writeResolvConf(nameserverIP string) error {
	filename := "/etc/resolv.conf"
	defaultNameserver := "127.0.0.53"
	content := fmt.Sprintf("nameserver %s\nnameserver %s\n", nameserverIP, defaultNameserver)
	err := os.WriteFile(filename, []byte(content), 0644)
	return err
}

func getHostIP() (string, error) {
	link, err := netlink.LinkByName("ens3")
	if err != nil {
		return "", err
	}
	addrList, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return "", err
	}
	if len(addrList) == 0 {
		return "", fmt.Errorf("no ip address found")
	}
	return addrList[0].IP.String(), nil
}

func (p *Proxy) syncLoop(ctx context.Context, wg *sync.WaitGroup,
	serviceUpdates <-chan *types.ServiceUpdate, dnsUpdates <-chan *types.DNSUpdate) {
	defer wg.Done()
	log.Printf("Sync loop started.")
	for {
		if !p.syncLoopIteration(ctx, serviceUpdates, dnsUpdates) {
			break
		}
	}
	log.Printf("Sync loop ended.")
	p.DoCleanUp()
}

func (p *Proxy) syncLoopIteration(ctx context.Context,
	updateCh <-chan *types.ServiceUpdate, dnsCh <-chan *types.DNSUpdate) bool {
	select {
	case update, ok := <-updateCh:
		if !ok {
			log.Printf("Service update channel closed.")
			return false
		}
		switch update.Op {
		case types.ServiceAdd:
			p.HandleServiceAdditions(update.Updates)
		case types.ServiceDelete:
			p.HandleServiceDeletions(update.Updates)
		case types.ServiceEndpointsUpdate:
			p.HandleServiceEndpointsUpdate(update.Updates)
		}
	case update, ok := <-dnsCh:
		if !ok {
			log.Printf("DNS update channel closed.")
			return false
		}
		switch update.Op {
		case types.DNSAdd:
			p.HandleDNSAdditions(update.DNS)
		case types.DNSDelete:
			p.HandleDNSDeletions(update.DNS)
		}
	case <-ctx.Done():
		return false
	}
	return true
}

func (p *Proxy) HandleDNSAdditions(dns []*v1.DNS) {
	log.Println("Handling dns additions...")
	for _, d := range dns {
		log.Printf("Adding dns %s.", d.Name)
		// step 1: 新建nginx配置文件 /etc/nginx/conf.d/<host>.conf
		err := p.addNginxConfig(d)
		if err != nil {
			log.Printf("Failed to write nginx config: %v", err)
			continue
		}
		// step 2: nginx -s reload
		err = exec.Command("nginx", "-s", "reload").Run()
		if err != nil {
			log.Printf("Failed to reload nginx: %v", err)
			continue
		}
		// step 3: 更新coredns配置文件 /etc/coredns/hosts
		err = p.addCorednsHosts(d)
		if err != nil {
			log.Printf("Failed to write coredns hosts: %v", err)
			continue
		}
		log.Printf("DNS %s added.", d.Name)
	}
}

func (p *Proxy) addCorednsHosts(dns *v1.DNS) error {
	filename := "/etc/coredns/hosts"
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	// 检查重复
	hosts := make(map[string]struct{})
	content := ""
	for scanner.Scan() {
		line := scanner.Text()
		host := strings.Fields(line)[1]
		hosts[host] = struct{}{}
		content += line + "\n"
	}
	err = file.Close()
	if err != nil {
		return err
	}
	for _, rule := range dns.Spec.Rules {
		if _, ok := hosts[rule.Host]; ok {
			log.Printf("Host %s already exists in coredns hosts.", rule.Host)
		} else {
			content += fmt.Sprintf("%s %s\n", p.hostIP, rule.Host)
		}
	}
	err = os.WriteFile(filename, []byte(content), 0644)
	return err
}

func (p *Proxy) addNginxConfig(dns *v1.DNS) error {
	confFmt := "server{listen 80;server_name %v;%v}"
	locationFmt := "location %v {proxy_pass http://%v:%v/;}"
	filenameFmt := "/etc/nginx/conf.d/%v.conf"
	for _, rule := range dns.Spec.Rules {
		filename := fmt.Sprintf(filenameFmt, rule.Host)
		// 检查是否存在重名host
		if _, err := os.Stat(filename); err == nil || !os.IsNotExist(err) {
			return fmt.Errorf("nginx config file %s already exists", filename)
		}
		location := ""
		for _, path := range rule.Paths {
			svcFullName := dns.Namespace + "_" + path.Backend.Service.Name
			if clusterIP, ok := p.serviceCache[svcFullName]; ok {
				singleLocation := fmt.Sprintf(locationFmt, path.Path, clusterIP, path.Backend.Service.Port)
				location += singleLocation
			} else {
				return fmt.Errorf("service %s not found in cache", svcFullName)
			}
		}
		if location == "" {
			return fmt.Errorf("no service found for dns %v", dns.Name)
		}
		conf := fmt.Sprintf(confFmt, rule.Host, location)
		err := os.WriteFile(filename, []byte(conf), 0644)
		if err != nil {
			return fmt.Errorf("failed to write nginx config: %v", err)
		}
	}
	return nil
}

func (p *Proxy) HandleDNSDeletions(dns []*v1.DNS) {
	log.Println("Handling dns deletions...")
	for _, d := range dns {
		log.Printf("Deleting dns %s.", d.Name)
		// step 1: 更新coredns配置文件 /etc/coredns/hosts
		err := p.deleteCorednsHosts(d)
		if err != nil {
			log.Printf("Failed to delete coredns hosts: %v", err)
			continue
		}
		// step 2: 删除nginx配置文件 /etc/nginx/conf.d/<host>.conf
		err = p.deleteNginxConfig(d)
		if err != nil {
			log.Printf("Failed to delete nginx config: %v", err)
			continue
		}
		// step 3: nginx -s reload
		err = exec.Command("nginx", "-s", "reload").Run()
		if err != nil {
			log.Printf("Failed to reload nginx: %v", err)
			continue
		}
		log.Printf("DNS %s deleted.", d.Name)
	}
}

func (p *Proxy) deleteCorednsHosts(dns *v1.DNS) error {
	filename := "/etc/coredns/hosts"
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	hostsToDelete := make(map[string]struct{})
	for _, rule := range dns.Spec.Rules {
		hostsToDelete[rule.Host] = struct{}{}
	}
	content := ""
	for scanner.Scan() {
		line := scanner.Text()
		host := strings.Fields(line)[1]
		if _, ok := hostsToDelete[host]; !ok {
			content += line + "\n"
		}
	}
	err = file.Close()
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, []byte(content), 0644)
	return err
}

func (p *Proxy) addServiceNameDNS(service *v1.Service) error {
	filename := "/etc/coredns/hosts"
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	// 检查重复
	hosts := make(map[string]struct{})
	content := ""
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("invalid coredns hosts file: %s", line)
		}
		host := fields[1]
		hosts[host] = struct{}{}
		content += line + "\n"
	}
	err = file.Close()
	if err != nil {
		return err
	}
	if _, ok := hosts[service.Name]; ok {
		return fmt.Errorf("service name %s already exists in coredns hosts", service.Name)
	} else if service.Spec.ClusterIP == "" {
		return fmt.Errorf("service %s has no cluster ip", service.Name)
	} else {
		content += fmt.Sprintf("%s %s\n", service.Spec.ClusterIP, service.Name)
	}
	err = os.WriteFile(filename, []byte(content), 0644)
	return err
}

func (p *Proxy) deleteServiceNameDNS(service *v1.Service) error {
	filename := "/etc/coredns/hosts"
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	content := ""
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("invalid coredns hosts file: %s", line)
		}
		host := fields[1]
		if host != service.Name {
			content += line + "\n"
		} else if fields[0] != service.Spec.ClusterIP {
			return fmt.Errorf("cluster ip mismatch for service %s", service.Name)
		}
	}
	err = file.Close()
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, []byte(content), 0644)
	return err
}

func (p *Proxy) deleteNginxConfig(dns *v1.DNS) error {
	filenameFmt := "/etc/nginx/conf.d/%v.conf"
	for _, rule := range dns.Spec.Rules {
		filename := fmt.Sprintf(filenameFmt, rule.Host)
		err := os.Remove(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) HandleServiceAdditions(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service additions...")
	for _, update := range updates {
		log.Printf("Adding service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		// 反向映射：targetPort -> port / nodePort
		reverseSvcPortMap := make(map[int32]int32)
		reverseNodePortMap := make(map[int32]int32)
		isNodePort := svc.Spec.Type == v1.ServiceTypeNodePort
		for _, svcPort := range svc.Spec.Ports {
			err := p.ipvs.AddVirtual(vip, uint16(svcPort.Port), svcPort.Protocol, true)
			if err != nil {
				log.Printf("Failed to add virtual server: %v", err)
				continue
			}
			if isNodePort {
				err = p.ipvs.AddVirtual(p.hostIP, uint16(svcPort.NodePort), svcPort.Protocol, false)
				if err != nil {
					log.Printf("Failed to add virtual server: %v", err)
					continue
				}
				reverseNodePortMap[svcPort.TargetPort] = svcPort.NodePort
			}
			reverseSvcPortMap[svcPort.TargetPort] = svcPort.Port
		}
		for _, endpoint := range update.EndpointAdditions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reverseSvcPortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
				if nodePort, ok := reverseNodePortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(p.hostIP, uint16(nodePort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
			}
		}
		p.serviceCache[utils.GetObjectFullName(&svc.ObjectMeta)] = vip
		err := p.addServiceNameDNS(svc)
		if err != nil {
			log.Printf("Failed to add service name dns: %v", err)
		}
		log.Printf("Service %s added.", svc.Name)
	}
}

func (p *Proxy) HandleServiceDeletions(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service deletions...")
	for _, update := range updates {
		log.Printf("Deleting service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		isNodePort := svc.Spec.Type == v1.ServiceTypeNodePort
		for _, svcPort := range svc.Spec.Ports {
			err := p.ipvs.DeleteVirtual(vip, uint16(svcPort.Port), svcPort.Protocol, true)
			if err != nil {
				log.Printf("Failed to delete virtual server: %v", err)
			}
			if isNodePort {
				err = p.ipvs.DeleteVirtual(p.hostIP, uint16(svcPort.NodePort), svcPort.Protocol, false)
				if err != nil {
					log.Printf("Failed to delete virtual server: %v", err)
				}
			}
		}
		delete(p.serviceCache, utils.GetObjectFullName(&svc.ObjectMeta))
		log.Printf("Service %s deleted.", svc.Name)
	}
}

func (p *Proxy) HandleServiceEndpointsUpdate(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service endpoints update...")
	for _, update := range updates {
		log.Printf("Updating service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		reverseSvcPortMap := make(map[int32]int32)
		reverseNodePortMap := make(map[int32]int32)
		isNodePort := svc.Spec.Type == v1.ServiceTypeNodePort
		for _, svcPort := range svc.Spec.Ports {
			reverseSvcPortMap[svcPort.TargetPort] = svcPort.Port
			if isNodePort {
				reverseNodePortMap[svcPort.TargetPort] = svcPort.NodePort
			}
		}
		for _, endpoint := range update.EndpointDeletions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reverseSvcPortMap[endpointPort.Port]; ok {
					err := p.ipvs.DeleteRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to delete route: %v", err)
					}
				}
				if nodePort, ok := reverseNodePortMap[endpointPort.Port]; ok {
					err := p.ipvs.DeleteRoute(p.hostIP, uint16(nodePort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to delete route: %v", err)
					}
				}
			}
		}
		for _, endpoint := range update.EndpointAdditions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reverseSvcPortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
				if nodePort, ok := reverseNodePortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(p.hostIP, uint16(nodePort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
			}
		}
		log.Printf("Service %s updated.", svc.Name)
	}
}

func cleanConfigFiles() {
	nginxFiles, err := os.ReadDir("/etc/nginx/conf.d")
	if err != nil {
		log.Printf("Failed to read nginx conf.d: %v", err)
	}
	for _, file := range nginxFiles {
		err = os.Remove("/etc/nginx/conf.d/" + file.Name())
		if err != nil {
			log.Printf("Failed to remove nginx config file: %v", err)
		}
	}
	err = os.Truncate("/etc/coredns/hosts", 0)
	if err != nil {
		log.Printf("Failed to truncate coredns hosts: %v", err)
	}
}

func (p *Proxy) DoCleanUp() {
	err := p.ipvs.Clear()
	if err != nil {
		log.Printf("Failed to clear ipvs: %v", err)
	}
	cleanConfigFiles()
	log.Printf("Proxy cleanup done.")
}
