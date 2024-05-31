package main

import (
	"fmt"
	"minikubernetes/pkg/kubelet/app"
	"net"
	"os"
)

func usage() {
	fmt.Println("usage: kubelet -j|--join <apiServerIP>")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		usage()
	}
	if os.Args[1] != "-j" && os.Args[1] != "--join" {
		usage()
	}
	ip := os.Args[2]
	if net.ParseIP(ip) == nil {
		fmt.Printf("invalid ip: %s\n", ip)
		os.Exit(1)
	}
	kubeletServer, err := app.NewKubeletServer(ip)
	// kubeletServer, err := app.NewKubeletServer("10.119.12.123")
	if err != nil {
		return
	}
	kubeletServer.Run()
}
