package pilot

import (
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"minikubernetes/pkg/utils"
	"strconv"
	"strings"
	"time"
)

type Pilot interface {
	Start() error
	SyncLoop() error
}

type pilot struct {
	client kubeclient.Client
}

func NewPilot(apiServerIP string) Pilot {
	manager := &pilot{}
	manager.client = kubeclient.NewClient(apiServerIP)
	return manager
}

func (p *pilot) Start() error {
	err := p.SyncLoop()
	if err != nil {
		return err
	}
	return nil
}

func (p *pilot) SyncLoop() error {
	for {
		err := p.syncLoopIteration()
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(5 * time.Second)
	}

}

func (p *pilot) doRollingUpdate(rollingUpdates []*v1.RollingUpdate, serviceMap map[string]*v1.Service, pods []*v1.Pod) {
	for _, ru := range rollingUpdates {
		if ru.Status.Phase != v1.RollingUpdatePending {
			continue
		}
		serviceFullName := ru.Namespace + "_" + ru.Spec.ServiceRef
		if service, ok := serviceMap[serviceFullName]; ok {
			var servicePods []*v1.Pod
			for _, pod := range pods {
				if !p.isSelectorMatched(pod, service) {
					continue
				}
				if pod.Status.Phase != v1.PodRunning {
					continue
				}
				if pod.Status.PodIP == "" {
					continue
				}
				servicePods = append(servicePods, pod)
			}
			if len(servicePods) != 0 {
				go p.rollingUpdateWorkerLoop(ru, service, servicePods)
			}
		}
	}
}

func (p *pilot) rollingUpdateWorkerLoop(rollingUpdate *v1.RollingUpdate, service *v1.Service, pods []*v1.Pod) {
	updateNum := len(pods) - int(rollingUpdate.Spec.MinimumAlive)
	if updateNum <= 0 {
		log.Printf("cannot update now")
		return
	}
	rollingUpdate.Status.Phase = v1.RollingUpdateRunning
	err := p.client.UpdateRollingUpdateStatus(rollingUpdate.Name, rollingUpdate.Namespace, &rollingUpdate.Status)
	if err != nil {
		log.Printf("update rolling update status failed: %v", err)
		return
	}
	subsetBlocked := &v1.Subset{
		TypeMeta: v1.TypeMeta{
			Kind:       "Subset",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      service.Name + "-blocked",
			Namespace: rollingUpdate.Namespace,
		},
		Spec: v1.SubsetSpec{
			Pods: nil,
		},
	}
	subsetAvailable := &v1.Subset{
		TypeMeta: v1.TypeMeta{
			Kind:       "Subset",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      service.Name + "-available",
			Namespace: rollingUpdate.Namespace,
		},
		Spec: v1.SubsetSpec{
			Pods: nil,
		},
	}
	err = p.client.AddSubset(subsetBlocked)
	if err != nil {
		log.Printf("add blocked subset failed: %v", err)
		return
	}
	err = p.client.AddSubset(subsetAvailable)
	if err != nil {
		log.Printf("add available subset failed: %v", err)
		return
	}
	vs := &v1.VirtualService{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      service.Name + "-rolling-update",
			Namespace: rollingUpdate.Namespace,
		},
		Spec: v1.VirtualServiceSpec{
			ServiceRef: rollingUpdate.Spec.ServiceRef,
			Port:       rollingUpdate.Spec.Port,
			Subsets: []v1.VirtualServiceSubset{
				{
					Name:   service.Name + "-available",
					Weight: &[]int32{1}[0],
				},
				{
					Name:   service.Name + "-blocked",
					Weight: &[]int32{0}[0],
				},
			},
		},
	}
	err = p.client.AddVirtualService(vs)
	if err != nil {
		log.Printf("add virtual service failed: %v", err)
		return
	}
	for i := 0; i < len(pods); i += updateNum {
		j := i + updateNum
		if j > len(pods) {
			j = len(pods)
		}
		blockedPodNames := make([]string, 0)
		availablePodNames := make([]string, 0)
		for k := 0; k < len(pods); k++ {
			if k >= i && k < j {
				blockedPodNames = append(blockedPodNames, pods[k].Name)
			} else {
				availablePodNames = append(availablePodNames, pods[k].Name)
			}
		}
		//for _, blockedPodName := range blockedPodNames {
		//	err = p.client.DeletePod(blockedPodName, rollingUpdate.Namespace)
		//	if err != nil {
		//		log.Printf("delete pod failed: %v", err)
		//	}
		//}
		subsetBlocked.Spec.Pods = blockedPodNames
		subsetAvailable.Spec.Pods = availablePodNames
		err = p.client.AddSubset(subsetBlocked)
		if err != nil {
			log.Printf("update blocked subset failed: %v", err)
		}
		err = p.client.AddSubset(subsetAvailable)
		if err != nil {
			log.Printf("update available subset failed: %v", err)
		}
		time.Sleep(time.Duration(rollingUpdate.Spec.Interval) * time.Second)
		//blockedPods := pods[i:j]
		//for _, blockedPod := range blockedPods {
		//	err = p.client.AddPod(*blockedPod)
		//	if err != nil {
		//		log.Printf("create pod failed: %v", err)
		//	}
		//}
	}
	rollingUpdate.Status.Phase = v1.RollingUpdateFinished
	err = p.client.UpdateRollingUpdateStatus(rollingUpdate.Name, rollingUpdate.Namespace, &rollingUpdate.Status)
	if err != nil {
		log.Printf("update rolling update status failed: %v", err)
	}
	err = p.client.DeleteVirtualService(vs)
	if err != nil {
		log.Printf("delete virtual service failed: %v", err)
	}
	err = p.client.DeleteSubset(subsetBlocked)
	if err != nil {
		log.Printf("delete blocked subset failed: %v", err)
	}
	err = p.client.DeleteSubset(subsetAvailable)
	if err != nil {
		log.Printf("delete available subset failed: %v", err)
	}
}

func (p *pilot) syncLoopIteration() error {
	var sideCarMap v1.SidecarMapping = make(v1.SidecarMapping)
	pods, err := p.client.GetAllPods()
	if err != nil {
		return err
	}
	services, err := p.client.GetAllServices()
	if err != nil {
		return err
	}
	virtualServices, err := p.client.GetAllVirtualServices()
	if err != nil {
		return err
	}
	rollingUpdates, err := p.client.GetAllRollingUpdates()
	if err != nil {
		return err
	}

	servicesMap := p.makeFullMap(services)
	endpointMap := p.getEndpoints(pods, services)
	markedMap := p.makeMarkedMap(services)

	p.doRollingUpdate(rollingUpdates, servicesMap, pods)

	for _, vs := range virtualServices {
		serviceNamespace := vs.Namespace
		serviceName := vs.Spec.ServiceRef
		serviceFullName := serviceNamespace + "_" + serviceName
		serviceUID := servicesMap[serviceFullName].UID
		serviceAndEndpointIncludingPodName := endpointMap[serviceUID]

		markedMap[p.getFullPort(serviceUID, vs.Spec.Port)] = true
		var sidecarEndpoints []v1.SidecarEndpoints
		if vs.Spec.Subsets[0].URL != nil {

			for _, sbs := range vs.Spec.Subsets {

				subset, err := p.client.GetSubsetByName(sbs.Name, vs.Namespace)
				if err != nil {
					return err
				}

				url := sbs.URL
				for _, podName := range subset.Spec.Pods {
					var singleEndPoints []v1.SingleEndpoint
					endpoints := serviceAndEndpointIncludingPodName.EndpointsMapWithPodName[podName]
					for _, endpoint := range endpoints.Ports {
						singleEndPoints = append(singleEndPoints, v1.SingleEndpoint{
							IP:         endpoints.IP,
							TargetPort: endpoint.Port,
						})
					}
					sidecarEndpoints = append(sidecarEndpoints, v1.SidecarEndpoints{
						URL:       url,
						Weight:    nil,
						Endpoints: singleEndPoints,
					})
				}

			}

		} else {
			var VsToSbsWeight []int32
			var SbsToPodNum []int32
			for _, sbs := range vs.Spec.Subsets {

				subset, err := p.client.GetSubsetByName(sbs.Name, vs.Namespace)
				if err != nil {
					return err
				}

				VsToSbsWeight = append(VsToSbsWeight, *sbs.Weight)
				SbsToPodNum = append(SbsToPodNum, int32(len(subset.Spec.Pods)))
			}

			RealWeight := p.calculate(VsToSbsWeight, SbsToPodNum)

			for i, sbs := range vs.Spec.Subsets {

				subset, err := p.client.GetSubsetByName(sbs.Name, vs.Namespace)
				if err != nil {
					return err
				}

				for _, podName := range subset.Spec.Pods {
					endpoints := serviceAndEndpointIncludingPodName.EndpointsMapWithPodName[podName]
					var singleEndPoints []v1.SingleEndpoint
					for _, endpoint := range endpoints.Ports {
						singleEndPoints = append(singleEndPoints, v1.SingleEndpoint{
							IP:         endpoints.IP,
							TargetPort: endpoint.Port,
						})
					}
					sidecarEndpoints = append(sidecarEndpoints, v1.SidecarEndpoints{
						URL:       nil,
						Weight:    &(RealWeight[i]),
						Endpoints: singleEndPoints,
					})
				}

			}

		}

		stringIP := serviceAndEndpointIncludingPodName.Service.Spec.ClusterIP + fmt.Sprintf(":%d", vs.Spec.Port)
		sideCarMap[stringIP] = sidecarEndpoints

	}

	var waitedServiceUidAndPorts map[v1.UID][]int32 = make(map[v1.UID][]int32)
	for serviceUIDAndPort, isDone := range markedMap {
		if !isDone {
			//println(serviceUIDAndPort)
			uid, port := p.splitFullPort(serviceUIDAndPort)
			waitedServiceUidAndPorts[uid] = append(waitedServiceUidAndPorts[uid], port)
			//println(uid)
			//println(fmt.Sprintf(":%d", port))
		}
	}
	newMap := make(map[string]*v1.ServiceAndEndpoints)
	for uid, portSs := range waitedServiceUidAndPorts {
		endpointsWithPodName := make(map[string]v1.Endpoint)
		service := endpointMap[uid].Service
		for _, port_i := range portSs {
			var port v1.ServicePort
			for _, portt := range service.Spec.Ports {
				if port_i == portt.Port {
					port = portt
				}
			}

			for _, pod := range pods {
				if !p.isSelectorMatched(pod, service) {
					continue
				}
				if pod.Status.Phase != v1.PodRunning {
					continue
				}
				if pod.Status.PodIP == "" {
					continue
				}

				var ports []v1.EndpointPort

			SearchLoop:
				for _, container := range pod.Spec.Containers {
					for _, containerPort := range container.Ports {
						if containerPort.ContainerPort == port.TargetPort && containerPort.Protocol == port.Protocol {
							ports = append(ports, v1.EndpointPort{
								Port:     port.TargetPort,
								Protocol: port.Protocol,
							})
							break SearchLoop
						}
					}
				}

				endpointsWithPodName[pod.Name] = v1.Endpoint{
					IP:    pod.Status.PodIP,
					Ports: ports,
				}
			}
			newMap[p.getFullPort(service.UID, port_i)] = &v1.ServiceAndEndpoints{
				Service:                 service,
				EndpointsMapWithPodName: endpointsWithPodName,
			}
		}
	}

	defaultWeight := int32(1)

	for port, serviceAndEndpoints := range newMap {
		_, port_i := p.splitFullPort(port)
		stringIP := serviceAndEndpoints.Service.Spec.ClusterIP + fmt.Sprintf(":%d", port_i)
		var sidecarEndpoints []v1.SidecarEndpoints
		for _, endpoint := range serviceAndEndpoints.EndpointsMapWithPodName {
			var singleEndPoints []v1.SingleEndpoint
			for _, endPointPort := range endpoint.Ports {
				singleEndPoints = append(singleEndPoints, v1.SingleEndpoint{
					IP:         endpoint.IP,
					TargetPort: endPointPort.Port,
				})
			}
			if singleEndPoints == nil {
				continue
			}
			sidecarEndpoints = append(sidecarEndpoints, v1.SidecarEndpoints{
				Weight:    &defaultWeight,
				URL:       nil,
				Endpoints: singleEndPoints,
			})
		}
		sideCarMap[stringIP] = sidecarEndpoints

	}
	err = p.client.AddSidecarMapping(sideCarMap)
	if err != nil {
		return err
	}

	//for stringIP, sidecarEndpoints := range sideCarMap {
	//	fmt.Println("IP IS :" + stringIP)
	//	for i, sidecarEndpoint := range sidecarEndpoints {
	//		fmt.Println(i)
	//		fmt.Println(*sidecarEndpoint.Weight)
	//		for _, endpoint := range sidecarEndpoint.Endpoints {
	//			fmt.Println(endpoint.IP + ": " + fmt.Sprintf("%d", endpoint.TargetPort))
	//		}
	//	}
	//	fmt.Println(" ")
	//}
	//return nil

	return nil
}

func (p *pilot) getEndpoints(pods []*v1.Pod, services []*v1.Service) map[v1.UID]*v1.ServiceAndEndpoints {
	newMap := make(map[v1.UID]*v1.ServiceAndEndpoints)
	for _, service := range services {
		endpointsWithPodName := make(map[string]v1.Endpoint)
		for _, pod := range pods {
			if !p.isSelectorMatched(pod, service) {
				continue
			}
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			if pod.Status.PodIP == "" {
				continue
			}

			var ports []v1.EndpointPort
			for _, port := range service.Spec.Ports {
			SearchLoop:
				for _, container := range pod.Spec.Containers {
					for _, containerPort := range container.Ports {
						if containerPort.ContainerPort == port.TargetPort && containerPort.Protocol == port.Protocol {
							ports = append(ports, v1.EndpointPort{
								Port:     port.TargetPort,
								Protocol: port.Protocol,
							})
							break SearchLoop
						}
					}
				}
			}

			endpointsWithPodName[pod.Name] = v1.Endpoint{
				IP:    pod.Status.PodIP,
				Ports: ports,
			}
		}
		newMap[service.ObjectMeta.UID] = &v1.ServiceAndEndpoints{
			Service:                 service,
			EndpointsMapWithPodName: endpointsWithPodName,
		}
	}
	return newMap
}

func (p *pilot) isSelectorMatched(pod *v1.Pod, svc *v1.Service) bool {
	for key, value := range svc.Spec.Selector {
		if label, ok := pod.Labels[key]; !ok || label != value {
			return false
		}
	}
	return true
}

func (p *pilot) makeFullMap(services []*v1.Service) map[string]*v1.Service {
	maps := make(map[string]*v1.Service)

	for _, service := range services {
		maps[utils.GetObjectFullName(&service.ObjectMeta)] = service
	}

	return maps
}

func (p *pilot) makeUIDMap(services []*v1.Service) map[v1.UID]*v1.Service {
	uidMap := make(map[v1.UID]*v1.Service)
	for _, service := range services {
		uidMap[service.ObjectMeta.UID] = service
	}
	return uidMap
}

func (p *pilot) makeMarkedMap(services []*v1.Service) map[string]bool {
	maps := make(map[string]bool)
	for _, service := range services {
		for _, port := range service.Spec.Ports {
			maps[p.getFullPort(service.UID, port.Port)] = false
		}
	}
	return maps
}

func (p *pilot) calculate(VsToSbsWeight []int32, SbsToPodNum []int32) []int32 {
	var result []int32
	for i, VTSW := range VsToSbsWeight {
		temp := VTSW
		for j, STPW := range SbsToPodNum {
			if i != j && STPW != 0 {
				temp = temp * STPW
			}
		}
		result = append(result, temp)
	}
	return result
}

func (p *pilot) getFullPort(uid v1.UID, port int32) string {
	return string(uid) + "_" + fmt.Sprintf("%d", port)
}

func (p *pilot) splitFullPort(fp string) (uid v1.UID, port int32) {
	parts := strings.Split(fp, "_")
	uid = v1.UID(parts[0])
	portInt, _ := strconv.Atoi(parts[1])
	port = int32(portInt)
	return uid, port
}
