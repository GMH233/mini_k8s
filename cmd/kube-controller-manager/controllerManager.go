package main

import (
	"minikubernetes/pkg/controller"
)

func main() {
	cm := controller.NewControllerManager("192.168.1.10")
	cm.Run()
}
