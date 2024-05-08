package route

import (
	"fmt"
	"github.com/moby/ipvs"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"net"
	"os/exec"
)

type IPVS interface {
	Init() error
	AddVirtual(vip string, port uint16, protocol v1.Protocol) error
	AddRoute(vip string, vport uint16, rip string, rport uint16, protocol v1.Protocol) error
	DeleteVirtual(vip string, port uint16, protocol v1.Protocol) error
	DeleteRoute(vip string, vport uint16, rip string, rport uint16, protocol v1.Protocol) error
	Clear() error
}

type basicIPVS struct {
	handle *ipvs.Handle
}

func NewIPVS() (IPVS, error) {
	ret := &basicIPVS{}
	handle, err := ipvs.New("")
	if err != nil {
		return nil, err
	}
	ret.handle = handle
	return ret, nil
}

func createDummy() error {
	dummy := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: "minik8s-dummy",
		},
	}
	err := netlink.LinkAdd(dummy)
	return err
}

func deleteDummy() error {
	link, err := netlink.LinkByName("minik8s-dummy")
	if err != nil {
		return err
	}
	err = netlink.LinkDel(link)
	return err
}

func addAddrToDummy(cidr string) error {
	link, err := netlink.LinkByName("minik8s-dummy")
	if err != nil {
		return err
	}
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	ipNet.IP = ip
	addr := &netlink.Addr{
		IPNet: ipNet,
	}
	err = netlink.AddrAdd(link, addr)
	return err
}

func delAddrFromDummy(cidr string) error {
	link, err := netlink.LinkByName("minik8s-dummy")
	if err != nil {
		return err
	}
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	ipNet.IP = ip
	addr := &netlink.Addr{
		IPNet: ipNet,
	}
	err = netlink.AddrDel(link, addr)
	return err
}

func (b *basicIPVS) Init() error {
	// 为了能使用ipvs的完整功能，使用jcloud镜像时，至少执行以下命令：
	// modprobe br_netfilter
	// ip link add dev minik8s-dummy type dummy
	// sysctl --write net.bridge.bridge-nf-call-iptables=1
	// sysctl --write net.ipv4.ip_forward=1
	// sysctl --write net.ipv4.vs.conntrack=1
	err := exec.Command("modprobe", "br_netfilter").Run()
	if err != nil {
		return fmt.Errorf("modprobe br_netfilter failed: %v", err)
	}
	err = createDummy()
	if err != nil {
		if err.Error() == "file exists" {
			log.Println("dummy already exists")
		} else {
			return fmt.Errorf("create dummy failed: %v", err)
		}
	}
	err = exec.Command("sysctl", "--write", "net.bridge.bridge-nf-call-iptables=1").Run()
	if err != nil {
		return fmt.Errorf("sysctl --write net.bridge.bridge-nf-call-iptables=1 failed: %v", err)
	}
	err = exec.Command("sysctl", "--write", "net.ipv4.ip_forward=1").Run()
	if err != nil {
		return fmt.Errorf("sysctl --write net.ipv4.ip_forward=1 failed: %v", err)
	}
	err = exec.Command("sysctl", "--write", "net.ipv4.vs.conntrack=1").Run()
	if err != nil {
		return fmt.Errorf("sysctl --write net.ipv4.vs.conntrack=1 failed: %v", err)
	}
	return nil
}

func (b *basicIPVS) AddVirtual(vip string, port uint16, protocol v1.Protocol) error {
	addr := net.ParseIP(vip)
	if addr == nil {
		return fmt.Errorf("invalid ip address: %s", vip)
	}
	svc := &ipvs.Service{
		Address:       addr,
		Protocol:      protocol2Unix(protocol),
		Port:          port,
		AddressFamily: unix.AF_INET,
		SchedName:     ipvs.RoundRobin,
	}
	err := b.handle.NewService(svc)
	if err != nil {
		return err
	}
	// ip addr add
	err = addAddrToDummy(vip + "/32")
	if err != nil {
		if err.Error() == "file exists" {
			log.Println("ip addr already exists")
		} else {
			return err
		}
	}
	return nil
}

func (b *basicIPVS) AddRoute(vip string, vport uint16, rip string, rport uint16, protocol v1.Protocol) error {
	vaddr := net.ParseIP(vip)
	if vaddr == nil {
		return fmt.Errorf("invalid ip address: %s", vip)
	}
	raddr := net.ParseIP(rip)
	if raddr == nil {
		return fmt.Errorf("invalid ip address: %s", rip)
	}
	svc := &ipvs.Service{
		Address:       vaddr,
		Protocol:      protocol2Unix(protocol),
		Port:          vport,
		AddressFamily: unix.AF_INET,
	}
	dest := &ipvs.Destination{
		Address:         raddr,
		Port:            rport,
		AddressFamily:   unix.AF_INET,
		Weight:          1,
		ConnectionFlags: ipvs.ConnectionFlagMasq,
	}
	err := b.handle.NewDestination(svc, dest)
	return err
}

func (b *basicIPVS) DeleteVirtual(vip string, port uint16, protocol v1.Protocol) error {
	addr := net.ParseIP(vip)
	if addr == nil {
		return fmt.Errorf("invalid ip address: %s", vip)
	}
	svc := &ipvs.Service{
		Address:       addr,
		Protocol:      protocol2Unix(protocol),
		Port:          port,
		AddressFamily: unix.AF_INET,
	}
	err := b.handle.DelService(svc)
	if err != nil {
		return err
	}
	err = delAddrFromDummy(vip + "/32")
	return err
}

func (b *basicIPVS) DeleteRoute(vip string, vport uint16, rip string, rport uint16, protocol v1.Protocol) error {
	vaddr := net.ParseIP(vip)
	if vaddr == nil {
		return fmt.Errorf("invalid ip address: %s", vip)
	}
	raddr := net.ParseIP(rip)
	if raddr == nil {
		return fmt.Errorf("invalid ip address: %s", rip)
	}
	svc := &ipvs.Service{
		Address:       vaddr,
		Protocol:      protocol2Unix(protocol),
		Port:          vport,
		AddressFamily: unix.AF_INET,
	}
	dest := &ipvs.Destination{
		Address:       raddr,
		Port:          rport,
		AddressFamily: unix.AF_INET,
	}
	err := b.handle.DelDestination(svc, dest)
	return err
}

func (b *basicIPVS) Clear() error {
	err := b.handle.Flush()
	if err != nil {
		return err
	}
	return deleteDummy()
}

func protocol2Unix(protocol v1.Protocol) uint16 {
	switch protocol {
	case v1.ProtocolTCP:
		return unix.IPPROTO_TCP
	case v1.ProtocolUDP:
		return unix.IPPROTO_UDP
	default:
		return unix.IPPROTO_TCP
	}
}
