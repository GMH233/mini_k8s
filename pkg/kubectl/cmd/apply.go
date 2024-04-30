package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"minikubernetes/pkg/kubectl/client"
	"minikubernetes/pkg/kubectl/utils"
	"os"
)

func init() {
	rootCmd.AddCommand(applyCmd)
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration to a resource by yaml file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			apply(arg)
		}
	},
}

func apply(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonBytes, err := utils.YAML2JSON(content)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(jsonBytes))
	err = client.NewKubectlClient(apiServerIP).AddPod(jsonBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
}
