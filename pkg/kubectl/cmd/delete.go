package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"minikubernetes/pkg/kubectl/client"
)

func init() {
	rootCmd.AddCommand(deleteCommand)
}

var deleteCommand = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 2 {
			if args[0] == "pod" {
				deletePod(args[1])
			}
		}
	},
}

func deletePod(podName string) {
	err := client.NewKubectlClient(apiServerIP).DeletePod(podName, "default")
	if err != nil {
		fmt.Println(err)
		return
	}
}
