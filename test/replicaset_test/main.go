package main

import (
	"fmt"
	"minikubernetes/pkg/controller/replicaset"
)

func main() {
	rsm := replicaset.NewReplicasetManager("192.168.1.10")
	err := rsm.RunRSC()
	if err != nil {
		fmt.Println(err)
	}
}
