package main

import "minikubernetes/pkg/kubeproxy/route"

func main() {
	ipvs, err := route.NewIPVS()
	if err != nil {
		panic(err)
	}
	err = ipvs.Init()
	if err != nil {
		panic(err)
	}
	err = ipvs.AddVirtual("100.100.100.10", 30080)
	if err != nil {
		panic(err)
	}
	err = ipvs.AddRoute("100.100.100.10", 30080, "10.32.0.1", 80)
	if err != nil {
		panic(err)
	}
	err = ipvs.DeleteRoute("100.100.100.10", 30080, "10.32.0.1", 80)
	if err != nil {
		panic(err)
	}
	err = ipvs.DeleteVirtual("100.100.100.10", 30080)
	if err != nil {
		panic(err)
	}
}
