package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	logLevel    int
	directory   string
	pattern     string
	template    string
	sendTo      string
	sendFrom    string
	smtpServer  string
	elkURL      string
	elkUsername string
	elkPassword string
	file        string
	secret      string
	tag         string
	index       string
	noinput     bool
)

var RootCmd = &cobra.Command{
	Use:   "xops",
	Short: "XCover operations CLI",
}

// Execute is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RootCmd.PersistentFlags().IntVarP(&logLevel, "log", "", 3, "Set log level")
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
