package cmd

import (
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Commands for working with services",
}

func init() {
	RootCmd.AddCommand(serviceCmd)
}
