package main

import "minikubernetes/pkg/kubeapiserver/app"

/* This is the starting interface of apiserver in main */

func main() {
	kubeApiServer, err := app.NewKubeApiServer()
	if err != nil {
		return
	}
	kubeApiServer.Run()
}
