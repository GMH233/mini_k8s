package main

import (
	"minikubernetes/pkg/controller/podautoscaler"
)

func main() {
	hc := podautoscaler.NewHorizonalController("192.168.1.10")
	hc.Run()

}
