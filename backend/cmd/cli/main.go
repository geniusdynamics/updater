package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ns8-updater",
	Short: "A general-purpose dependency updater",
	Long:  `A CLI and server to scan for and update dependencies of various types.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no subcommand is given
		fmt.Println("Use 'ns8-updater help' for more information.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}