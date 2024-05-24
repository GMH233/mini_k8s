package main

import (
	"log"
	envoyinit "minikubernetes/pkg/microservice/envoy/init"
)

func main() {
	e, err := envoyinit.NewEnvoyInit()
	if err != nil {
		log.Fatalf("Failed to create envoy: %v", err)
	}
	err = e.Init()
	if err != nil {
		log.Fatalf("Failed to init envoy: %v", err)
	}
	log.Println("Envoy init success.")
}
