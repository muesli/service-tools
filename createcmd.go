package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type CreateOptions struct {
	Executable  string
	Description string

	WorkingDirectory string
	User             string
	Group            string

	RestartSec uint
	Restart    string

	After    string
	WantedBy string
}

var (
	createOpts = CreateOptions{}

	createCmd = &cobra.Command{
		Use:   "create <executable> <description>",
		Short: "creates a new Unit file",
		Long:  `The create command creates a new systemd Unit file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if len(args) >= 1 {
				createOpts.Executable = args[0]
			} else {
				createOpts.Executable, err = readString("Which executable do you want to create a service for", true)
				if err != nil {
					return fmt.Errorf("create needs an executable to create a service for")
				}
			}

			stat, err := os.Stat(createOpts.Executable)
			if os.IsNotExist(err) {
				return fmt.Errorf("Can't create service: no such file")
			}
			if stat.IsDir() {
				return fmt.Errorf("Can't create service: target is a directory")
			}
			if stat.Mode()&0111 == 0 {
				return fmt.Errorf("Can't create service: target is not executable")
			}

			if len(args) >= 2 {
				createOpts.Description = args[1]
			} else {
				createOpts.Description, _ = readString("Description", true)
			}
			if len(createOpts.Description) == 0 {
				return fmt.Errorf("create needs a description for this service")
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
