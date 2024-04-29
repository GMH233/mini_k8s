package main

import "minikubernetes/pkg/kubelet/app"

func main() {
	kubeletServer, err := app.NewKubeletServer("10.119.12.123")
	if err != nil {
		return
	}
	kubeletServer.Run()
}
