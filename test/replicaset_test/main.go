package main

import (
	"fmt"
	"minikubernetes/pkg/controller"
)

func main() {
	rsm := controller.NewReplicasetManager()
	err := rsm.RunRSC()
	if err != nil {
		fmt.Println(err)
	}
}
