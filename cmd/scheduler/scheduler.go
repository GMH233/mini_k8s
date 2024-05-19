package main

import (
	"fmt"
	scheduler2 "minikubernetes/pkg/scheduler"
	"os"
)

func main() {
	var policy string
	if len(os.Args) < 2 {
		policy = "Round_Policy"
	} else {
		policy = os.Args[1]
	}

	switch policy {
	case "Round_Policy":
		break
	case "Random_Policy":
		break
	default:
		fmt.Println("Invalid policy")
		os.Exit(1)
	}
	scheduler := scheduler2.NewScheduler("10.119.12.123", policy)
	scheduler.Run()
}
