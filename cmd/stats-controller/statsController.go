package main

import (
	"minikubernetes/pkg/controller/stats"
)

func main() {
	manager := stats.NewStatsManager("192.168.1.10")
	err := manager.RunStatsC()
	if err != nil {
		return
	}
}
