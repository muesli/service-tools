package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-systemd/unit"
	"github.com/rivo/tview"
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
	RestartSec      string
	TimeoutStartSec string
	TimeoutStopSec  string

	After    string
	WantedBy string
}

var (
	createOpts = CreateOptions{}
	types      = Strings{"simple", "forking", "oneshot", "dbus", "notify", "idle"}
	restarts   = Strings{"no", "always", "on-success", "on-failure", "on-abnormal", "on-abort", "on-watchdog"}

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
			if !types.Contains(createOpts.Type) {
				return fmt.Errorf("No such service type: %s", createOpts.Type)
			}
			createOpts.Restart = strings.ToLower(createOpts.Restart)
			if !restarts.Contains(createOpts.Restart) {
				return fmt.Errorf("No such restart type: %s", createOpts.Restart)
			}

			if len(args) >= 1 {
				createOpts.Exec = args[0]
			}
			if len(args) >= 2 {
				createOpts.Description = args[1]
			}
			if len(args) >= 3 {
				createOpts.After = args[2]
			}
			if len(args) >= 4 {
				createOpts.WantedBy = args[3]

				validate()
				return executeCreate()
			}
			app := tview.NewApplication()
			form := tview.NewForm().
				AddInputField("Description:", createOpts.Description, 40, nil, func(s string) {
					createOpts.Description = s
				}).
				AddDropDown("Type:", types, types.IndexOf(createOpts.Type), func(s string, i int) {
					createOpts.Type = s
				}).
				AddInputField("Exec on start:", createOpts.Exec, 40, nil, func(s string) {
					createOpts.Exec = s
				}).
				AddInputField("Exec on stop:", createOpts.ExecStop, 40, nil, func(s string) {
					createOpts.ExecStop = s
				}).
				AddInputField("Exec on reload:", createOpts.ExecReload, 40, nil, func(s string) {
					createOpts.ExecReload = s
				}).
				AddDropDown("Restarts on:", restarts, restarts.IndexOf(createOpts.Restart), func(s string, i int) {
					createOpts.Restart = s
				}).
				AddDropDown("Start after target:", ts.Strings(), Strings(ts.Strings()).IndexOf(createOpts.After), func(s string, i int) {
					createOpts.After = s
				}).
				AddDropDown("Wanted by target:", ts.Strings(), Strings(ts.Strings()).IndexOf(createOpts.WantedBy), func(s string, i int) {
					createOpts.WantedBy = s
				})

			var apperr error
			form.
				AddButton("Create", func() {
					app.Stop()
					if err := validate(); err != nil {
						apperr = err
					} else {
						executeCreate()
					}
				}).
				AddButton("Cancel", func() {
					app.Stop()
				})

			form.SetBorder(true).SetTitle("Create new service").SetTitleAlign(tview.AlignCenter)
			if err := app.SetRoot(form, true).Run(); err != nil {
				return err
			}
			return apperr
		},
	}
)

func validate() error {
	ts, err := targets()
	if err != nil {
		return fmt.Errorf("Can't find systemd targets: %s", err)
	}

	// Exec check
	if len(strings.TrimSpace(createOpts.Exec)) == 0 {
		return fmt.Errorf("Need an executable to create a service for")
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

	// Description check
	if len(createOpts.Description) == 0 {
		return fmt.Errorf("Description for this service can't be empty")
	}

	// Target checks
	if len(createOpts.After) > 0 && !ts.Contains(createOpts.After) {
		return fmt.Errorf("Could not create service: no such target")
	}
	if len(createOpts.WantedBy) > 0 && !ts.Contains(createOpts.WantedBy) {
		return fmt.Errorf("Could not create service: no such target")
	}

	return nil
}

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
		&unit.UnitOption{"Service", "RestartSec", createOpts.RestartSec},
		&unit.UnitOption{"Service", "TimeoutStartSec", createOpts.TimeoutStartSec},
		&unit.UnitOption{"Service", "TimeoutStopSec", createOpts.TimeoutStopSec},

		&unit.UnitOption{"Install", "WantedBy", createOpts.WantedBy},
	}

	r := unit.Serialize(stripEmptyOptions(u))
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
	createCmd.PersistentFlags().StringVarP(&createOpts.RestartSec, "restartsec", "s", "", "How many seconds between restarts")
	createCmd.PersistentFlags().StringVar(&createOpts.TimeoutStartSec, "timeoutstartsec", "", "How many seconds to wait for a startup")
	createCmd.PersistentFlags().StringVar(&createOpts.TimeoutStopSec, "timeoutstopsec", "", "How many seconds to wait when stoping a service")

	createCmd.PersistentFlags().StringVarP(&createOpts.After, "after", "a", "", "Target after which the service will be started")
	createCmd.PersistentFlags().StringVarP(&createOpts.WantedBy, "wantedby", "b", "", "This service is wanted by this target")

	RootCmd.AddCommand(createCmd)
}
