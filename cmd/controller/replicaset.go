package main

import "minikubernetes/pkg/controller"

func main() {
	rsc := controller.NewReplicasetManager("10.119.12.123")
	err := rsc.RunRSC()
	if err != nil {
		return
	}
}
