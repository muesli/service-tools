package main

import (
	"os"

	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:           "service-monitor",
		Short:         "service-monitor monitors systemd Units",
		Long:          "service-monitor is a convenient little tool to monitor systemd Units",
		SilenceErrors: false,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logsCmd.RunE(cmd, args)
		},
	}

	apperr error
	app    = tview.NewApplication()
	menu   = NewMenu(app)
	search string
)

func main() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
