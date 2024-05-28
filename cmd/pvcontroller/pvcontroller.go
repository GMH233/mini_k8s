package main

import (
	"log"
	"minikubernetes/pkg/controller/pv"
)

func main()  {
	c, err := pv.NewPVController("192.168.1.10")
	if err != nil {
		log.Fatalf("Failed to create PV controller: %v", err)
	}
	c.Run()
}
