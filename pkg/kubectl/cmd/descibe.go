package cmd

// 导出原始的json文件

import (
	"fmt"
	"minikubernetes/pkg/kubeclient"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func init() {

	rootCmd.AddCommand(describeCommand)
}

var describeCommand = &cobra.Command{
	Use:   "describe",
	Short: "Describe resources",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			if args[0] == "pod" {
				describePod(args[1], "default")
			}
			if args[0] == "service" {
				describeService(args[1], "default")
			}
			if args[0] == "hpa" {
				describeHPA(args[1], "default")
			}
			if args[0] == "replicaset" {
				describeReplicaSet(args[1], "default")
			}
			if args[0] == "virtualservice" {
				describeVirtualService(args[1], "default")
			}
			if args[0] == "subsets" {
				describeSubset(args[1], "default")
			}
			if args[0] == "dns" {
				describeDNS(args[1], "default")
			}

		} else if len(args) == 1 {
			// 指定namespace和name
			namespace, _ := cmd.Flags().GetString("namespace")
			name, _ := cmd.Flags().GetString("name")
			if namespace == "" {
				namespace = "default"
			}
			if name == "" {
				fmt.Println("Usage: kubectl describe [pod|service|hpa|replicaset] -np [namespace] -n [name]")
				return
			}
			switch args[0] {
			case "pod":
				describePod(name, namespace)
			case "service":
				describeService(name, namespace)
			case "hpa":
				describeHPA(name, namespace)
			case "replicaset":
				describeReplicaSet(name, namespace)
			case "virtualservice":
				describeVirtualService(name, namespace)
			case "subsets":
				describeSubset(name, namespace)
			case "dns":
				describeDNS(name, namespace)
			}

		}
	},
}

func describePod(name, namespace string) {
	pod, err := kubeclient.NewClient(apiServerIP).GetPod(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Phase", "IP"})
	table.Append([]string{pod.Name, pod.Namespace, string(pod.Status.Phase), pod.Status.PodIP})
	table.Render()
}

func describeService(name, namespace string) {
	service, err := kubeclient.NewClient(apiServerIP).GetService(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "ClusterIP", "Ports"})
	table.Append([]string{service.Name, service.Namespace, service.Spec.ClusterIP, fmt.Sprintf("%v", service.Spec.Ports)})
	table.Render()
}

func describeHPA(name, namespace string) {
	hpa, err := kubeclient.NewClient(apiServerIP).GetHPAScaler(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "MinReplicas", "MaxReplicas", "Metrics"})
	table.Append([]string{hpa.Name, hpa.Namespace, fmt.Sprintf("%v", hpa.Spec.MinReplicas), fmt.Sprintf("%v", hpa.Spec.MaxReplicas), fmt.Sprintf("%v", hpa.Spec.Metrics)})
	table.Render()
}

func describeReplicaSet(name, namespace string) {
	replicaSet, err := kubeclient.NewClient(apiServerIP).GetReplicaSet(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Replicas", "Selector"})
	table.Append([]string{replicaSet.Name, replicaSet.Namespace, fmt.Sprintf("%v", replicaSet.Spec.Replicas), fmt.Sprintf("%v", replicaSet.Spec.Selector)})
	table.Render()
}

func describeVirtualService(name, namespace string) {
	virtualService, err := kubeclient.NewClient(apiServerIP).GetVirtualService(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "ServiceRef", "Port", "Subsets"})
	table.Append([]string{virtualService.Name, virtualService.Namespace, virtualService.Spec.ServiceRef, fmt.Sprintf("%v", virtualService.Spec.Port), fmt.Sprintf("%v", virtualService.Spec.Subsets)})
	table.Render()
}

func describeSubset(name, namespace string) {
	subset, err := kubeclient.NewClient(apiServerIP).GetSubsetByName(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Labels", "Pods"})
	table.Append([]string{subset.Name, subset.Namespace, fmt.Sprintf("%v", subset.Labels), fmt.Sprintf("%v", subset.Spec.Pods)})
	table.Render()
}

func describeDNS(name, namespace string) {
	dns, err := kubeclient.NewClient(apiServerIP).GetDNS(name, namespace)
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Rules"})
	table.Append([]string{dns.Name, dns.Namespace, fmt.Sprintf("%v", dns.Spec.Rules)})
	table.Render()
}

// TODO 增加rolling update
