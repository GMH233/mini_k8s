package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
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
			fmt.Printf("Apply configuration from file: %s\n", arg)
		}
	},
}
