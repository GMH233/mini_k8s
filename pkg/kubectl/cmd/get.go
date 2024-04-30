package cmd

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"minikubernetes/pkg/kubectl/client"
	"os"
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
				getPods()
			}
		}
	},
}

func getPods() {
	pods, err := client.NewKubectlClient(apiServerIP).GetPods()
	if err != nil {
		fmt.Println(err)
		return
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Kind", "Name", "Namespace", "Phase", "IP"})
	for _, pod := range pods {
		table.Append([]string{"pod", pod.Name, pod.Namespace, string(pod.Status.Phase), pod.Status.PodIP})
	}
	table.Render()
}
