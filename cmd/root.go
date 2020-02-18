package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	logLevel int
)

var RootCmd = &cobra.Command{
	Use:   "k8sutil",
	Short: "Command-line utility for working with kubernetes resources",
}

// Execute is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RootCmd.PersistentFlags().IntVarP(&logLevel, "log", "", 3, "Set log level")
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
