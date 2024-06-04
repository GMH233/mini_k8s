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
			if args[0] == "pods" || args[0] == "pod" {
				getAllPods()
			}
			if args[0] == "nodes" || args[0] == "node" {
				getAllNodes()
			}
			if args[0] == "services" || args[0] == "service" {
				getAllServices()
			}
			if args[0] == "hpas" || args[0] == "hpa" {
				getAllHPAScalers()
			}
			if args[0] == "replicasets" || args[0] == "replicaset" {
				getAllReplicaSets()
			}
			if args[0] == "virtualservices" || args[0] == "virtualservice" {
				getAllVirtualServices()
			}
			if args[0] == "subsets" || args[0] == "subset" {
				getAllSubsets()
			}
			if args[0] == "dns" {
				getAllDNS()
			}
			if args[0] == "rollingupdates" || args[0] == "rollingupdate" {
				getAllRollingUpdate()
			}
			if args[0] == "pv" {
				getAllPVs()
			}
			if args[0] == "pvc" {
				getAllPVCs()
			}
		}
	},
}

func getAllPVCs() {
	pvcs, err := kubeclient.NewClient(apiServerIP).GetAllPVCs()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Request", "StorageClass", "Phase", "VolumeName"})
	for _, pvc := range pvcs {
		table.Append([]string{
			"PersistentVolumeClaim",
			pvc.Namespace,
			pvc.Name,
			pvc.Spec.Request,
			pvc.Spec.StorageClassName,
			string(pvc.Status.Phase),
			pvc.Status.VolumeName,
		})
	}
	table.Render()
}

func getAllPVs() {
	pvs, err := kubeclient.NewClient(apiServerIP).GetAllPVs()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Capacity", "StorageClass", "Phase", "Server", "Path", "ClaimName"})
	for _, pv := range pvs {
		table.Append([]string{"PersistentVolume",
			pv.Namespace,
			pv.Name,
			pv.Spec.Capacity,
			pv.Spec.StorageClassName,
			string(pv.Status.Phase),
			pv.Status.Server,
			pv.Status.Path,
			pv.Status.ClaimName,
		})
	}
	table.Render()
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
			subsetsFmtStr += fmt.Sprintf("%v\n", subset.Name)
		}
		subsetsFmtStr = strings.TrimSpace(subsetsFmtStr)

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
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Pods"})
	for _, subset := range subsets {
		podFmtStr := ""
		for _, pod := range subset.Spec.Pods {
			podFmtStr += fmt.Sprintf("%v\n", pod)
		}
		podFmtStr = strings.TrimSpace(podFmtStr)
		table.Append([]string{"subset", subset.Namespace, subset.Name, podFmtStr})
	}
	table.Render()
}

func getAllDNS() {
	dns, err := kubeclient.NewClient(apiServerIP).GetAllDNS()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Rules"})
	for _, dn := range dns {
		hostPath2NamePortStr := ""
		for _, rule := range dn.Spec.Rules {
			host := rule.Host
			for _, dnsPath := range rule.Paths {
				hostPath2NamePortStr += fmt.Sprintf("%v%v\n --> %v:%v\n", host, dnsPath.Path, dnsPath.Backend.Service.Name, dnsPath.Backend.Service.Port)
			}
		}
		if hostPath2NamePortStr != "" {
			hostPath2NamePortStr = strings.TrimSpace(hostPath2NamePortStr)
			table.Append([]string{"dns", dn.Namespace, dn.Name, hostPath2NamePortStr})
		} else {
			table.Append([]string{"dns", dn.Namespace, dn.Name, "N/A"})
		}
	}
	table.Render()
}

// TODO 展示rolling update
func getAllRollingUpdate() {
	rollingUpdates, err := kubeclient.NewClient(apiServerIP).GetAllRollingUpdates()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Namespace", "Name", "Status", "ServiceRef", "Port", "MinimumAlive", "Interval"})
	for _, rollingUpdate := range rollingUpdates {
		table.Append([]string{"rollingupdate", rollingUpdate.Namespace, rollingUpdate.Name, string(rollingUpdate.Status.Phase), rollingUpdate.Spec.ServiceRef, fmt.Sprint(rollingUpdate.Spec.Port), fmt.Sprint(rollingUpdate.Spec.MinimumAlive), fmt.Sprint(rollingUpdate.Spec.Interval)})
	}
	table.Render()

}
