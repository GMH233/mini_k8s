package main

import "minikubernetes/pkg/kubeproxy/app"

func main() {
	server, err := app.NewProxyServer("10.119.12.123")
	if err != nil {
		panic(err)
	}
	server.Run()
}
