package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"minikubernetes/pkg/kubectl/util"
	"net/http"
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
	jsonBytes, err := util.YAML2JSON(content)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(jsonBytes))
	pod, err := util.JSON2Pod(jsonBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	var namespace string
	if pod.Namespace == "" {
		namespace = "default"
	} else {
		namespace = pod.Namespace
	}
	// POST to API server
	url := fmt.Sprintf("%s/namespaces/%s/pods", apiUrl, namespace)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode != http.StatusCreated {
		fmt.Println("Error: ", resp.Status)
		return
	}
	fmt.Printf("Created Pod %v.", pod.Name)
}
