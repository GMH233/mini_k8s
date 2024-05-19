package main

import (
	"minikubernetes/pkg/controller"
)

func main() {
	manager := controller.NewReplicasetManager("10.119.12.123")
	err := manager.RunRSC()
	if err != nil {
		return
	}
}
