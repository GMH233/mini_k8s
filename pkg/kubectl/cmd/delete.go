package cmd

import (
	"encoding/json"
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"minikubernetes/pkg/kubectl/client"
	"minikubernetes/pkg/kubectl/utils"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	deleteCommand.Flags().StringP("file", "f", "", "YAML file to delete resources from")
	deleteCommand.Flags().StringP("namespace", "p", "default", "Namespace of the resources")
	deleteCommand.Flags().StringP("name", "n", "", "Name of the resources")
	rootCmd.AddCommand(deleteCommand)
}

var deleteCommand = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// 指定文件名
		filename, _ := cmd.Flags().GetString("file")
		if filename != "" {
			fmt.Println("Delete from file: ", filename)
			deleteFromYAML(filename)
			return
		}

		// 不从文件中删除
		// 直接指定名字，默认在default namespace下删除
		if len(args) == 2 {
			switch args[0] {
			case "pod":
				deletePod(args[1], "default")
			case "service":
				deleteService(args[1], "default")
			case "hpa":
				deleteHPA(args[1], "default")
			case "replicaset":
				deleteReplicaSet(args[1], "default")
			}
		} else if len(args) == 1 {
			// 指定namespace和name
			namespace, _ := cmd.Flags().GetString("namespace")
			name, _ := cmd.Flags().GetString("name")
			if namespace == "" {
				namespace = "default"
			}
			if name == "" {
				fmt.Println("Usage: kubectl delete [pod|service|hpa|replicaset] -np [namespace] -n [name]")
				return
			}
			switch args[0] {
			case "pod":
				deletePod(name, namespace)
			case "service":
				deleteService(name, namespace)
			case "hpa":
				deleteHPA(name, namespace)
			case "replicaset":
				deleteReplicaSet(name, namespace)
			}

		} else {
			fmt.Println("Invalid usage.")
		}
	},
}

func deleteFromYAML(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	kind := utils.GetKind(content)
	if kind == "" {
		fmt.Println("kind not found")
		return
	}
	jsonBytes, err := utils.YAML2JSON(content)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("get json: %v\n", string(jsonBytes))
	// 根据kind区分不同的资源
	switch kind {
	case "Pod":
		fmt.Println("Delete Pod")
		var podGenerated v1.Pod
		err := json.Unmarshal(jsonBytes, &podGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if podGenerated.Name == "" {
			fmt.Println("Pod name not found")
			return
		}
		if podGenerated.Namespace == "" {
			podGenerated.Namespace = "default"
		}
		deletePod(podGenerated.Name, podGenerated.Namespace)

		fmt.Println("Pod Deleted")

	case "Service":
		fmt.Println("Delete Service")
		var serviceGenerated v1.Service
		err := json.Unmarshal(jsonBytes, &serviceGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if serviceGenerated.Name == "" {
			fmt.Println("Service name not found")
			return
		}
		if serviceGenerated.Namespace == "" {
			serviceGenerated.Namespace = "default"
		}
		deleteService(serviceGenerated.Name, serviceGenerated.Namespace)
		fmt.Println("Service Deleted")

	case "ReplicaSet":
		fmt.Println("Delete ReplicaSet")
		var replicaSetGenerated v1.ReplicaSet
		err := json.Unmarshal(jsonBytes, &replicaSetGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if replicaSetGenerated.Name == "" {
			fmt.Println("ReplicaSet name not found")
			return
		}
		if replicaSetGenerated.Namespace == "" {
			replicaSetGenerated.Namespace = "default"
		}
		deleteReplicaSet(replicaSetGenerated.Name, replicaSetGenerated.Namespace)

		fmt.Println("ReplicaSet Deleted")

	case "HorizontalPodAutoscaler":
		fmt.Println("Delete HorizontalPodAutoscaler")
		var hpaGenerated v1.HorizontalPodAutoscaler
		err := json.Unmarshal(jsonBytes, &hpaGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if hpaGenerated.Name == "" {
			fmt.Println("HPA name not found")
			return
		}
		if hpaGenerated.Namespace == "" {
			hpaGenerated.Namespace = "default"
		}
		deleteHPA(hpaGenerated.Name, hpaGenerated.Namespace)
		fmt.Println("HorizontalPodAutoscaler Deleted")

	}
}

func deletePod(podName, nameSpace string) {
	err := client.NewKubectlClient(apiServerIP).DeletePod(podName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func deleteService(serviceName, nameSpace string) {
	err := kubeclient.NewClient(apiServerIP).DeleteService(serviceName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}
func deleteHPA(hpaName, nameSpace string) {
	err := kubeclient.NewClient(apiServerIP).DeleteHPAScaler(hpaName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func deleteReplicaSet(replicaSetName, nameSpace string) {

	err := kubeclient.NewClient(apiServerIP).DeleteReplicaSet(replicaSetName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}
