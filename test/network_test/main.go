package main

import (
	"fmt"
	"minikubernetes/pkg/kubelet/network"
)

func main() {
	IP1, err := network.Attach("9adb28a544e9")
	if err != nil {
		panic(err)
	}
	IP2, err := network.Attach("ed970ca4c658")
	if err != nil {
		panic(err)
	}
	fmt.Printf("IP1: %s, IP2: %s\n", IP1, IP2)
	err = network.Detach("9adb28a544e9")
	if err != nil {
		panic(err)
	}
	err = network.Detach("ed970ca4c658")
	if err != nil {
		panic(err)
	}
}
