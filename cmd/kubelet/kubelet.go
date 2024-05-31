package main

import (
	"encoding/json"
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubectl/utils"
	"minikubernetes/pkg/kubelet/app"
	"net"
	"os"
)

func usage() {
	fmt.Println("usage: kubelet -j|--join <apiServerIP> [-c|--config <configFile>]")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 && len(os.Args) != 5 {
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
	var node v1.Node
	if len(os.Args) == 5 {
		if os.Args[3] != "-c" && os.Args[3] != "--config" {
			usage()
		}
		filename := os.Args[4]
		yamlBytes, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("failed to read file %s: %v\n", filename, err)
			os.Exit(1)
		}
		jsonBytes, err := utils.YAML2JSON(yamlBytes)
		if err != nil {
			fmt.Printf("failed to parse yaml file\n")
			os.Exit(1)
		}
		err = json.Unmarshal(jsonBytes, &node)
		if err != nil {
			fmt.Printf("failed to parse yaml file\n")
			os.Exit(1)
		}
	}
	kubeletServer, err := app.NewKubeletServer(ip, &node)
	// kubeletServer, err := app.NewKubeletServer("10.119.12.123")
	if err != nil {
		return
	}
	kubeletServer.Run()
}
