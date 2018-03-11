package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/unit"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	Type        string
	Description string

	Exec          string
	ExecStartPre  string
	ExecStartPost string
	ExecReload    string
	ExecStop      string
	ExecStopPost  string

	WorkingDirectory string
	RootDirectory    string
	User             string
	Group            string

	Restart         string
	RestartSec      uint64
	TimeoutStartSec uint64
	TimeoutStopSec  uint64

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

			createOpts.Type = strings.ToLower(createOpts.Type)
			switch createOpts.Type {
			case "simple":
			case "forking":
			case "oneshot":
			case "dbus":
			case "notify":
			case "idle":
			default:
				return fmt.Errorf("No such service type: %s", createOpts.Type)
			}
			createOpts.Restart = strings.ToLower(createOpts.Restart)
			switch createOpts.Restart {
			case "no":
			case "always":
			case "on-success":
			case "on-failure":
			case "on-abnormal":
			case "on-abort":
			case "on-watchdog":
			default:
				return fmt.Errorf("No such service type: %s", createOpts.Type)
			}

			if len(args) >= 1 {
				createOpts.Exec = args[0]
			} else {
				createOpts.Exec, err = readString("Which executable do you want to create a service for", true)
				if err != nil {
					return fmt.Errorf("create needs an executable to create a service for")
				}
			}

			stat, err := os.Stat(createOpts.Exec)
			if os.IsNotExist(err) {
				return fmt.Errorf("Could not find executable: %s is not a file", createOpts.Exec)
			}
			if stat.IsDir() {
				return fmt.Errorf("Could not find executable: %s is a directory", createOpts.Exec)
			}
			if stat.Mode()&0111 == 0 {
				return fmt.Errorf("%s is not executable", createOpts.Exec)
			}

			if len(args) >= 2 {
				createOpts.Description = args[1]
			} else {
				createOpts.Description, _ = readString("Description", true)
			}
			if len(createOpts.Description) == 0 {
				return fmt.Errorf("Description for this service can't be empty")
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
				return fmt.Errorf("Could not create service: no such target")
			}

			if len(createOpts.WantedBy) == 0 {
				createOpts.WantedBy, _ = readString("Which target should this service be wanted by", true)
				if len(createOpts.WantedBy) == 0 {
					return fmt.Errorf("create needs a target which this service will be wanted by")
				}
			}
			if !ts.Contains(createOpts.WantedBy) {
				return fmt.Errorf("Could not create service: no such target")
			}

			return executeCreate()
		},
	}
)

func executeCreate() error {
	u := []*unit.UnitOption{
		&unit.UnitOption{"Unit", "Description", createOpts.Description},
		&unit.UnitOption{"Unit", "After", createOpts.After},

		&unit.UnitOption{"Service", "Type", createOpts.Type},
		&unit.UnitOption{"Service", "WorkingDirectory", createOpts.WorkingDirectory},
		&unit.UnitOption{"Service", "RootDirectory", createOpts.RootDirectory},

		&unit.UnitOption{"Service", "ExecStart", createOpts.Exec},
		&unit.UnitOption{"Service", "ExecStartPre", createOpts.ExecStartPre},
		&unit.UnitOption{"Service", "ExecStartPost", createOpts.ExecStartPost},
		&unit.UnitOption{"Service", "ExecReload", createOpts.ExecReload},
		&unit.UnitOption{"Service", "ExecStop", createOpts.ExecStop},
		&unit.UnitOption{"Service", "ExecStopPost", createOpts.ExecStopPost},

		&unit.UnitOption{"Service", "User", createOpts.User},
		&unit.UnitOption{"Service", "Group", createOpts.Group},
		&unit.UnitOption{"Service", "Restart", createOpts.Restart},
		&unit.UnitOption{"Service", "RestartSec", strconv.FormatUint(createOpts.RestartSec, 10)},
		&unit.UnitOption{"Service", "TimeoutStartSec", strconv.FormatUint(createOpts.TimeoutStartSec, 10)},
		&unit.UnitOption{"Service", "TimeoutStopSec", strconv.FormatUint(createOpts.TimeoutStopSec, 10)},

		&unit.UnitOption{"Install", "WantedBy", createOpts.WantedBy},
	}

	r := unit.Serialize(u)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("Encountered error while reading output: %v", err)
	}

	filename := filepath.Base(createOpts.Exec) + ".service"
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("Could not create file: %s", err)
	}
	defer f.Close()

	_, err = f.Write(b)
	if err != nil {
		return fmt.Errorf("Could not write to file: %s", err)
	}

	fmt.Printf("Generated Unit file: %s\n%s\n", filename, b)
	return nil
}

func init() {
	createCmd.PersistentFlags().StringVarP(&createOpts.Type, "type", "t", "simple", "Type of service (simple, forking, oneshot, dbus, notify or idle)")

	createCmd.PersistentFlags().StringVar(&createOpts.ExecStartPre, "execstartpre", "", "Executable to run before the service starts")
	createCmd.PersistentFlags().StringVar(&createOpts.ExecStartPost, "execstartpost", "", "Executable to run after the service started")
	createCmd.PersistentFlags().StringVar(&createOpts.ExecReload, "execreload", "", "Executable to run to reload the service")
	createCmd.PersistentFlags().StringVar(&createOpts.ExecStop, "execstop", "", "Executable to run to stop the service")
	createCmd.PersistentFlags().StringVar(&createOpts.ExecStopPost, "execstoppost", "", "Executable to run after the service stopped")

	createCmd.PersistentFlags().StringVarP(&createOpts.WorkingDirectory, "workingdir", "w", "", "Working-directory of the service")
	createCmd.PersistentFlags().StringVar(&createOpts.RootDirectory, "rootdir", "", "Root-directory of the service")
	createCmd.PersistentFlags().StringVarP(&createOpts.User, "user", "u", "root", "User to run service as")
	createCmd.PersistentFlags().StringVarP(&createOpts.Group, "group", "g", "root", "Group to run service as")

	createCmd.PersistentFlags().StringVarP(&createOpts.Restart, "restart", "r", "on-failure", "When to restart (no, always, on-success, on-failure, on-abnormal, on-abort or on-watchdog)")
	createCmd.PersistentFlags().Uint64VarP(&createOpts.RestartSec, "restartsec", "s", 5, "How many seconds between restarts")
	createCmd.PersistentFlags().Uint64Var(&createOpts.TimeoutStartSec, "timeoutstartsec", 0, "How many seconds to wait for a startup")
	createCmd.PersistentFlags().Uint64Var(&createOpts.TimeoutStopSec, "timeoutstopsec", 0, "How many seconds to wait when stoping a service")

	createCmd.PersistentFlags().StringVarP(&createOpts.After, "after", "a", "", "Target after which the service will be started")
	createCmd.PersistentFlags().StringVarP(&createOpts.WantedBy, "wantedby", "b", "", "This service is wanted by this target")

	RootCmd.AddCommand(createCmd)
}
