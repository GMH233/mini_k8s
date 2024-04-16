package main

import "minikubernetes/pkg/kubelet/app"

func main() {
	kubeletServer, err := app.NewKubeletServer("0.0.0.0")
	if err != nil {
		return
	}
	kubeletServer.Run()
}
