package main

import (
	"fmt"
	"minikubernetes/pkg/controller"
)

func main() {
	rsm := controller.NewReplicasetManager("10.119.12.123")
	err := rsm.RunRSC()
	if err != nil {
		fmt.Println(err)
	}
}
