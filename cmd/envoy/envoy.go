package main

import (
	"log"
	"minikubernetes/pkg/microservice/envoy"
)

func main() {
	e, err := envoy.NewEnvoy("10.119.12.123")
	if err != nil {
		log.Fatalf("Failed to create envoy: %v", err)
	}
	e.Run()
}
