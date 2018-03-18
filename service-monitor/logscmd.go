package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type LogModel struct {
	Name        string
	Description string
	Matches     []sdjournal.Match
}

var (
	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "monitor service logs",
		Long:  `The logs command starts an interactive service monitor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ts, err := services()
			if err != nil {
				return fmt.Errorf("Can't find systemd services: %s", err)
			}

			model := []LogModel{
				{
					Name:        "All",
					Description: "Everything in the log",
					Matches:     []sdjournal.Match{},
				},
				{
					Name:        "All Errors",
					Description: "All errors in the log",
					Matches: []sdjournal.Match{
						{
							Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
							Value: "3",
						},
					},
				},
				{
					Name:        "All Warnigns",
					Description: "All warnings in the log",
					Matches: []sdjournal.Match{
						{
							Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
							Value: "3",
						},
						{
							Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
							Value: "4",
						},
					},
				},
			}
			for _, service := range ts {
				model = append(model, LogModel{
					Name:        service.Name,
					Description: service.Description,
					Matches: []sdjournal.Match{
						{
							Field: sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT,
							Value: service.Name,
						},
					},
				})
			}

			var apperr error
			app := tview.NewApplication()

			list := tview.NewList()
			list.
				SetBorder(true).
				SetTitle("Logs")

			textView := tview.NewTextView()
			textView.
				SetDynamicColors(true).
				SetScrollable(true).
				SetChangedFunc(func() {
					app.Draw()
				}).
				SetDoneFunc(func(key tcell.Key) {
					app.SetFocus(list)
				})
			textView.
				SetBorder(true).
				SetTitle("Service Log")

			list.SetDoneFunc(func() {
				app.SetFocus(textView)
			})

			errLogView := tview.NewTextView()
			errLogView.
				SetDynamicColors(true).
				SetScrollable(true).
				SetChangedFunc(func() {
					app.Draw()
				}).
				SetDoneFunc(func(key tcell.Key) {
					app.SetFocus(list)
				})
			errLogView.
				SetBorder(true).
				SetTitle("Global Error Log")

			r := readLog([]sdjournal.Match{})
			errReader := readLog([]sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
					Value: "3",
				},
			})

			go func() {
				buf, _ := ioutil.ReadAll(errReader)
				errLogView.SetText(string(buf))
				errLogView.ScrollToEnd()
				for {
					line, err := errReader.ReadString('\n')
					if err != nil {
						return
					}
					errLogView.SetText(strings.TrimSpace(line))
					errLogView.ScrollToEnd()
				}
			}()

			list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
				u := model[index]
				textView.Clear()
				textView.SetTitle(u.Name)
				r = readLog(u.Matches)
				buf, _ := ioutil.ReadAll(r)
				textView.SetText(string(buf))
				textView.ScrollToEnd()
				app.SetFocus(textView)
			})

			for _, srv := range model {
				list.AddItem(srv.Name, srv.Description, 0, nil)
			}

			flex := tview.NewFlex().
				AddItem(list, 32, 1, true).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(textView, 0, 8, false).
					AddItem(errLogView, 8, 1, false), 0, 1, false)

			if err := app.SetRoot(flex, true).Run(); err != nil {
				return err
			}
			return apperr
		},
	}
)

func readLog(matches []sdjournal.Match) *bytes.Buffer {
	r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Since:   time.Duration(-60) * time.Minute * 24,
		Matches: matches,
	})

	if err != nil {
		panic(err)
	}
	if r == nil {
		panic("journal reader is nil")
	}

	defer r.Close()
	// r.Rewind()

	b := []byte{}
	buf := bytes.NewBuffer(b)

	// and follow the reader synchronously
	timeout := time.Duration(1) * time.Second
	if err = r.Follow(time.After(timeout), buf); err != sdjournal.ErrExpired {
		panic(err)
	}

	return buf
}

func init() {
	RootCmd.AddCommand(logsCmd)
}
