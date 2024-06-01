package cmd

import (
	"encoding/json"
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"minikubernetes/pkg/kubectl/utils"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	applyCmd.Flags().StringP("file", "f", "", "YAML file to apply resources from")
	rootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration to a resource by yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		filename, _ := cmd.Flags().GetString("file")
		if filename == "" {
			fmt.Println("Usage: kubectl apply -f [filename]")
			return
		}
		apply(filename)
	},
}

func apply(filename string) {
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
		fmt.Println("Apply Pod")
		var podGenerated v1.Pod
		err := json.Unmarshal(jsonBytes, &podGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyPod(podGenerated)
		fmt.Println("Pod Applied")

	case "Service":
		fmt.Println("Apply Service")
		var serviceGenerated v1.Service
		err := json.Unmarshal(jsonBytes, &serviceGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyService(serviceGenerated)
		fmt.Println("Service Applied")

	case "ReplicaSet":
		fmt.Println("Apply ReplicaSet")
		var replicaSetGenerated v1.ReplicaSet
		err := json.Unmarshal(jsonBytes, &replicaSetGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyReplicaSet(replicaSetGenerated)
		fmt.Println("ReplicaSet Applied")

	case "HorizontalPodAutoscaler":
		fmt.Println("Apply HorizontalPodAutoscaler")
		var hpaGenerated v1.HorizontalPodAutoscaler
		err := json.Unmarshal(jsonBytes, &hpaGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyHPAScaler(hpaGenerated)
		fmt.Println("HorizontalPodAutoscaler Applied")

	case "VirtualService":
		fmt.Println("Apply VirtualService")
		var virtualServiceGenerated v1.VirtualService
		err := json.Unmarshal(jsonBytes, &virtualServiceGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyVirtualService(&virtualServiceGenerated)
		fmt.Println("VirtualService Applied")

	case "Subset":
		fmt.Println("Apply Subset")
		var subsetGenerated v1.Subset
		err := json.Unmarshal(jsonBytes, &subsetGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applySubset(&subsetGenerated)
		fmt.Println("Subset Applied")

	case "DNS":
		fmt.Println("Apply DNS")
		var dnsGenerated v1.DNS
		err := json.Unmarshal(jsonBytes, &dnsGenerated)
		if err != nil {
			fmt.Println(err)
			return
		}
		applyDNS(dnsGenerated)
		fmt.Println("DNS Applied")
	}

}
func applyPod(pod v1.Pod) {
	err := kubeclient.NewClient(apiServerIP).AddPod(pod)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func applyService(service v1.Service) {
	err := kubeclient.NewClient(apiServerIP).AddService(service)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func applyReplicaSet(replicaSet v1.ReplicaSet) {
	err := kubeclient.NewClient(apiServerIP).AddReplicaSet(replicaSet)
	if err != nil {
		fmt.Println(err)
		return
	}
}
func applyHPAScaler(hpa v1.HorizontalPodAutoscaler) {
	err := kubeclient.NewClient(apiServerIP).AddHPAScaler(hpa)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func applyVirtualService(virtualService *v1.VirtualService) {
	err := kubeclient.NewClient(apiServerIP).AddVirtualService(virtualService)
	if err != nil {
		fmt.Println(err)
		return
	}
}
func applySubset(subset *v1.Subset) {
	err := kubeclient.NewClient(apiServerIP).AddSubset(subset)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func applyDNS(dns v1.DNS) {
	err := kubeclient.NewClient(apiServerIP).AddDNS(dns)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// TODO 增加rolling update的部署
// func applyRolingUpdate() {
// 	fmt.Println("Apply Rolling Update")
// }
