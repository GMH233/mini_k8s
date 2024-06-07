package main

import (
	"minikubernetes/pkg/controller/replicaset"
)

func main() {
	manager := replicaset.NewReplicasetManager("192.168.1.10")
	err := manager.RunRSC()
	if err != nil {
		return
	}
}
