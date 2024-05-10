package main

import (
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeproxy/route"
)

func main() {
	ipvs, err := route.NewIPVS()
	if err != nil {
		panic(err)
	}
	err = ipvs.Init()
	if err != nil {
		panic(err)
	}
	err = ipvs.AddVirtual("100.100.100.10", 30080, v1.ProtocolTCP, true)
	if err != nil {
		panic(err)
	}
	err = ipvs.AddRoute("100.100.100.10", 30080, "10.32.0.1", 80, v1.ProtocolTCP)
	if err != nil {
		panic(err)
	}
	err = ipvs.DeleteRoute("100.100.100.10", 30080, "10.32.0.1", 80, v1.ProtocolTCP)
	if err != nil {
		panic(err)
	}
	err = ipvs.DeleteVirtual("100.100.100.10", 30080, v1.ProtocolTCP, true)
	if err != nil {
		panic(err)
	}
}
