package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type CreateOptions struct {
	WorkingDirectory string
}

var (
	createOpts = CreateOptions{}

	createCmd = &cobra.Command{
		Use:   "create <executable>",
		Short: "creates a new Unit file",
		Long:  `The create command creates a new systemd Unit file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("create needs an executable to create a Unit file for")
			}
			return executeCreate()
		},
	}
)

func executeCreate() error {
	return nil
}

func init() {
	createCmd.PersistentFlags().StringVarP(&createOpts.WorkingDirectory, "workingdir", "w", "", "WorkingDirectory of the service")

	RootCmd.AddCommand(createCmd)
}
