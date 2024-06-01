package cmd

import (
	"fmt"
	"minikubernetes/pkg/kubeclient"
	"os"
	"strings"

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
			if args[0] == "virtualservices" {
				getAllVirtualServices()
			}
			if args[0] == "subsets" {
				getAllSubsets()
			}
			if args[0] == "dns" {
				getAllDNS()
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
	ipDetailsTable.SetHeader([]string{"Name/Service", "ClusterIP", "Port", "NodePort", "Endpoint", "Protocol"})
	// todo 显示service的port和endpoint
	for _, service := range services {
		ip := service.Spec.ClusterIP

		for _, svcPort := range service.Spec.Ports {

			sideCarMpKey := fmt.Sprintf("%v:%v", ip, svcPort.Port)
			allSidecarMap, err := kubeclient.NewClient(apiServerIP).GetSidecarMapping()
			if err != nil {
				fmt.Println(err)
				return
			}
			if sidecarEPList := allSidecarMap[sideCarMpKey]; sidecarEPList != nil {
				var epFmtStr string
				for _, sidecarEP := range sidecarEPList {
					for _, singleEP := range sidecarEP.Endpoints {
						epFmtStr += fmt.Sprintf("%v:%v\n", singleEP.IP, singleEP.TargetPort)
					}
				}
				epFmtStr = strings.TrimSpace(epFmtStr)
				if service.Spec.Type == "NodePort" {
					ipDetailsTable.Append([]string{service.Namespace + "/" + service.Name, ip, fmt.Sprint(svcPort.Port), fmt.Sprint(svcPort.NodePort), epFmtStr, fmt.Sprint(svcPort.Protocol)})
				} else {
					ipDetailsTable.Append([]string{service.Namespace + "/" + service.Name, ip, fmt.Sprint(svcPort.Port), "N/A", epFmtStr, fmt.Sprint(svcPort.Protocol)})
				}
			} else {
				if service.Spec.Type == "NodePort" {
					ipDetailsTable.Append([]string{service.Namespace + "/" + service.Name, ip, fmt.Sprint(svcPort.Port), fmt.Sprint(svcPort.NodePort), "N/A", fmt.Sprint(svcPort.Protocol)})
				} else {
					ipDetailsTable.Append([]string{service.Namespace + "/" + service.Name, ip, fmt.Sprint(svcPort.Port), "N/A", "N/A", fmt.Sprint(svcPort.Protocol)})
				}
			}

		}
	}
	ipDetailsTable.Render()

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

func getAllVirtualServices() {
	virtualservices, err := kubeclient.NewClient(apiServerIP).GetAllVirtualServices()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "ServiceRef", "Port", "Subsets"})
	for _, virtualservice := range virtualservices {
		subsetsFmtStr := ""
		for _, subset := range virtualservice.Spec.Subsets {
			subsetsFmtStr += fmt.Sprintf("%v\n", subset)
		}

		table.Append([]string{"virtualservice", virtualservice.Namespace, virtualservice.Name, virtualservice.Spec.ServiceRef, fmt.Sprint(virtualservice.Spec.Port), subsetsFmtStr})
	}
	table.Render()
}

func getAllSubsets() {
	subsets, err := kubeclient.NewClient(apiServerIP).GetAllSubsets()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Labels", "Pods"})
	for _, subset := range subsets {
		podFmtStr := ""
		for _, pod := range subset.Spec.Pods {
			podFmtStr += fmt.Sprintf("%v\n", pod)
		}
		labelsFmtStr := ""
		for k, v := range subset.Labels {
			labelsFmtStr += fmt.Sprintf("%v:%v\n", k, v)
		}
		table.Append([]string{"subset", subset.Namespace, subset.Name, labelsFmtStr, podFmtStr})
	}
	table.Render()
}

func getAllDNS() {
	// dns, err := kubeclient.NewClient(apiServerIP).GetAllDNS()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// table := tablewriter.NewWriter(os.Stdout)
	// table.SetHeader([]string{"Kind", "Name", "Namespace", "IP"})
	// for _, dn := range dns {
	// 	table.Append([]string{"dns", dn.Name, dn.Namespace, dn.Spec.IP})
	// }
	// table.Render()
}

// TODO 展示rolling update
func getAllRollingUpdate() {
	//
}
