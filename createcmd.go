package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/coreos/go-systemd/unit"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	Executable  string
	Description string

	WorkingDirectory string
	User             string
	Group            string

	RestartSec uint64
	Restart    string

	After    string
	WantedBy string
}

var (
	createOpts = CreateOptions{}

	createCmd = &cobra.Command{
		Use:   "create <executable> <description> <after> <wanted-by>",
		Short: "creates a new Unit file",
		Long:  `The create command creates a new systemd Unit file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ts, err := targets()
			if err != nil {
				return fmt.Errorf("Can't find systemd targets: %s", err)
			}

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

			if len(args) >= 4 {
				createOpts.WantedBy = args[3]
			}
			if len(args) >= 3 {
				createOpts.After = args[2]
			}
			if len(createOpts.After) == 0 || len(createOpts.WantedBy) == 0 {
				fmt.Println("Available targets:")
				for _, t := range ts {
					fmt.Printf("%s - %s\n", t.Name, t.Description)
				}
			}

			if len(createOpts.After) == 0 {
				createOpts.After, _ = readString("Start after target", true)
				if len(createOpts.After) == 0 {
					return fmt.Errorf("create needs a target after which this service will be started")
				}
			}
			if !ts.Contains(createOpts.After) {
				return fmt.Errorf("Can't create service: no such target")
			}

			if len(createOpts.WantedBy) == 0 {
				createOpts.WantedBy, _ = readString("Which target should this service be wanted by", true)
				if len(createOpts.WantedBy) == 0 {
					return fmt.Errorf("create needs a target which this service will be wanted by")
				}
			}
			if !ts.Contains(createOpts.WantedBy) {
				return fmt.Errorf("Can't create service: no such target")
			}

			return executeCreate()
		},
	}
)

func executeCreate() error {
	u := []*unit.UnitOption{
		&unit.UnitOption{"Unit", "Description", createOpts.Description},
		&unit.UnitOption{"Unit", "After", createOpts.After},

		&unit.UnitOption{"Service", "ExecStart", createOpts.Executable},
		&unit.UnitOption{"Service", "User", createOpts.User},
		&unit.UnitOption{"Service", "Group", createOpts.Group},
		&unit.UnitOption{"Service", "Restart", createOpts.Restart},
		&unit.UnitOption{"Service", "RestartSec", strconv.FormatUint(createOpts.RestartSec, 10)},

		&unit.UnitOption{"Install", "WantedBy", createOpts.WantedBy},
	}

	r := unit.Serialize(u)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("encountered error while reading output: %v", err)
	}

	filename := filepath.Base(createOpts.Executable) + ".service"
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Can't write file: %s", err)
	}
	defer f.Close()

	_, err = f.Write(b)
	if err != nil {
		return fmt.Errorf("Can't write to file: %s", err)
	}

	fmt.Printf("Generated Unit file: %s\n%s\n", filename, b)
	return nil
}

func init() {
	createCmd.PersistentFlags().StringVarP(&createOpts.WorkingDirectory, "workingdir", "w", "", "WorkingDirectory of the service")
	createCmd.PersistentFlags().StringVarP(&createOpts.User, "user", "u", "root", "User to run service as")
	createCmd.PersistentFlags().StringVarP(&createOpts.Group, "group", "g", "root", "Group to run service as")

	createCmd.PersistentFlags().StringVarP(&createOpts.Restart, "restart", "r", "on-failure", "When to restart the service")
	createCmd.PersistentFlags().Uint64VarP(&createOpts.RestartSec, "restartsec", "s", 5, "How many seconds between restarts")

	createCmd.PersistentFlags().StringVarP(&createOpts.After, "after", "a", "", "Target after which the service will be started")
	createCmd.PersistentFlags().StringVarP(&createOpts.WantedBy, "wantedby", "b", "", "This service is wanted by this target")

	RootCmd.AddCommand(createCmd)
}
