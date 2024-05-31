package cmd

import (
	"fmt"
	"minikubernetes/pkg/kubeclient"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCommand)
}

var getCommand = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			if args[0] == "pods" {
				getAllPods()
			}
			if args[0] == "nodes" {
				getAllNodes()
			}
			if args[0] == "services" {
				getAllServices()
			}
			if args[0] == "hpas" {
				getAllHPAScalers()
			}
			if args[0] == "replicasets" {
				getAllReplicaSets()
			}
		}
	},
}

func getAllPods() {
	pods, err := kubeclient.NewClient(apiServerIP).GetAllPods()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Phase", "IP"})
	for _, pod := range pods {
		table.Append([]string{"pod", pod.Namespace, pod.Name, string(pod.Status.Phase), pod.Status.PodIP})
	}
	table.Render()
}

func getAllNodes() {
	nodes, err := kubeclient.NewClient(apiServerIP).GetAllNodes()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Name", "IP"})
	for _, node := range nodes {
		table.Append([]string{"node", node.Name, node.Status.Address})
	}
	table.Render()

}

func getAllServices() {
	services, err := kubeclient.NewClient(apiServerIP).GetAllServices()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "ClusterIP"})
	for _, service := range services {
		table.Append([]string{"service", service.Namespace, service.Name, service.Spec.ClusterIP})
	}
	table.Render()

	ipDetailsTable := tablewriter.NewWriter(os.Stdout)
	ipDetailsTable.SetHeader([]string{"Name/Service", "ClusterIP", "Port", "Endpoint", "Protocol"})
	// for _, service := range services {
	// 	for _, port := range service.Spec.Ports {
	// 		ipDetailsTable.Append([]string{service.Name, service.Spec.ClusterIP, fmt.Sprint(port.Port), fmt.Sprint(port.NodePort), port.Protocol})
	// 	}
	// }

}
func getAllHPAScalers() {
	hpas, err := kubeclient.NewClient(apiServerIP).GetAllHPAScalers()

	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Min_Replicas", "Max_Replicas", "Current_Replicas"})
	for _, hpa := range hpas {
		// 找到相对应的rps的replica数量

		rps, err := kubeclient.NewClient(apiServerIP).GetReplicaSet(hpa.Spec.ScaleTargetRef.Name, hpa.Namespace)
		if err != nil {
			fmt.Println(err)
			return
		}
		currentReps := rps.Spec.Replicas

		table.Append([]string{"hpa", hpa.Namespace, hpa.Name, fmt.Sprint(hpa.Spec.MinReplicas), fmt.Sprint(hpa.Spec.MaxReplicas), fmt.Sprint(currentReps)})
	}
	table.Render()
}

func getAllReplicaSets() {
	replicasets, err := kubeclient.NewClient(apiServerIP).GetAllReplicaSets()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Replicas"})
	for _, replicaset := range replicasets {
		table.Append([]string{"replicaset", replicaset.Namespace, replicaset.Name, fmt.Sprint(replicaset.Spec.Replicas)})
	}
	table.Render()
}
