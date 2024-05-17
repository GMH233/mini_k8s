package utils

import (
	"fmt"
	"github.com/vishvananda/netlink"
)

func GetHostIP() (string, error) {
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
