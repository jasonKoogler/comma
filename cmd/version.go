package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	version = "0.1.0"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the Comma version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Comma version %s\n", version)
	},
}
