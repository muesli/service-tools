package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:           "service-generator",
		Short:         "service-generator creates systemd Unit files",
		Long:          "service-generator is a convenient little tool to create systemd Unit files",
		SilenceErrors: false,
		SilenceUsage:  true,
	}
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
