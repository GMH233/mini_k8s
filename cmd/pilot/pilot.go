package main

import (
	"fmt"
	"minikubernetes/pkg/microservice/pilot"
)

func main() {
	rsm := pilot.NewPilot("192.168.1.10")
	err := rsm.Start()
	if err != nil {
		fmt.Println(err)
	}
}
