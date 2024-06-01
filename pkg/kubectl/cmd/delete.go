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
			case "virtualservice":
				deleteVirtualService(args[1], "default")
			case "subset":
				deleteSubset(args[1], "default")
			case "dns":
				deleteDNS(args[1], "default")

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
			case "virtualservice":
				deleteVirtualService(name, namespace)
			case "subsets":
				deleteSubset(name, namespace)
			case "dns":
				deleteDNS(name, namespace)
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

	case "VirtualService":
		fmt.Println("Delete VirtualService")
		var virtualServiceGenerated v1.VirtualService
		err := json.Unmarshal(jsonBytes, &virtualServiceGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if virtualServiceGenerated.Name == "" {
			fmt.Println("VirtualService name not found")
			return
		}
		if virtualServiceGenerated.Namespace == "" {
			virtualServiceGenerated.Namespace = "default"
		}
		deleteVirtualService(virtualServiceGenerated.Name, virtualServiceGenerated.Namespace)
		fmt.Println("VirtualService Deleted")

	case "Subset":
		fmt.Println("Delete Subset")
		var subsetGenerated v1.Subset
		err := json.Unmarshal(jsonBytes, &subsetGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if subsetGenerated.Name == "" {
			fmt.Println("Subset name not found")
			return
		}
		if subsetGenerated.Namespace == "" {
			subsetGenerated.Namespace = "default"
		}
		deleteSubset(subsetGenerated.Name, subsetGenerated.Namespace)
		fmt.Println("Subset Deleted")
	case "DNS":
		fmt.Println("Delete DNS")
		var dnsGenerated v1.DNS
		err := json.Unmarshal(jsonBytes, &dnsGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		if dnsGenerated.Name == "" {
			fmt.Println("DNS name not found")
			return
		}
		if dnsGenerated.Namespace == "" {
			dnsGenerated.Namespace = "default"
		}
		deleteDNS(dnsGenerated.Name, dnsGenerated.Namespace)
		fmt.Println("DNS Deleted")
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

func deleteVirtualService(vsName, nameSpace string) {
	err := kubeclient.NewClient(apiServerIP).DeleteVirtualServiceByNameNp(vsName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func deleteSubset(subsetName, nameSpace string) {
	err := kubeclient.NewClient(apiServerIP).DeleteSubsetByNameNp(subsetName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func deleteDNS(dnsName, nameSpace string) {
	err := kubeclient.NewClient(apiServerIP).DeleteDNS(dnsName, nameSpace)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// TODO: 增加RolliingUpdate的删除
// func deleteRollingUpdate(rollingUpdateName, nameSpace string) {
// 	err := kubeclient.NewClient(apiServerIP).DeleteRollingUpdate(rollingUpdateName, nameSpace)
// }
