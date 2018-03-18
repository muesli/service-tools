package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:           "service-monitor",
		Short:         "service-monitor monitors systemd Units",
		Long:          "service-monitor is a convenient little tool to monitor systemd Units",
		SilenceErrors: false,
		SilenceUsage:  true,
	}
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
