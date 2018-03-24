package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var (
	logsCmd = &cobra.Command{
		Use:   "logs",
		Short: "monitor service logs",
		Long:  `The logs command starts an interactive service monitor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			form, err := logsForm()
			if err != nil {
				return err
			}

			if err = app.SetRoot(form, true).Run(); err != nil {
				return err
			}
			return apperr
		},
	}
)

func logsForm() (tview.Primitive, error) {
	var pipe *LogPipe
	activeOnly := true
	filter := logLevelFilter(6)

	list := NewServicesView()
	logView := tview.NewTextView()
	errLogView := tview.NewTextView()
	errLogView.ScrollToEnd()

	list.
		SetDoneFunc(func() {
			app.SetFocus(logView)
		})

	err := list.loadModel(true, activeOnly)
	if err != nil {
		return nil, err
	}

	logView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			var next tview.Primitive = errLogView
			switch key {
			case tcell.KeyBacktab:
				fallthrough
			case tcell.KeyEscape:
				next = list
			}
			app.SetFocus(next)
		})
	logView.
		SetBorder(true).
		SetTitle("Log")

	errLogView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			var next tview.Primitive = list
			switch key {
			case tcell.KeyBacktab:
				next = logView
			}
			app.SetFocus(next)
		})
	errLogView.
		SetBorder(true).
		SetTitle("Global Error Log")

	errReader := logPipe([]sdjournal.Match{}, logLevelFilter(3))
	go pipeReader(errReader, errLogView)

	list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		selectLog(pipe, list.Model[index], filter, logView)
	})

	flex := tview.NewFlex().
		AddItem(list, 32, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(logView, 0, 8, false).
			AddItem(errLogView, 8, 1, false), 0, 1, false)

	logLevelDropDown := tview.NewList()
	logLevelDropDown.
		SetBorder(true).
		SetTitle("Log-level")
	logLevelDialog := tview.NewFlex().
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(logLevelDropDown, 18, 1, true).
			AddItem(tview.NewBox(), 0, 1, false), 30, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)

	pages := tview.NewPages()
	pages.AddPage("flex", flex, true, true)
	pages.AddPage("dropdown_loglevel", logLevelDialog, true, false)

	menuPages := tview.NewPages()
	searchInput := tview.NewInputField()
	searchInput.
		SetLabel("Search for: ").
		SetFieldWidth(40).
		SetAcceptanceFunc(nil).
		SetDoneFunc(func(key tcell.Key) {
			search = searchInput.GetText()
			selectLog(pipe, list.Model[list.GetCurrentItem()], filter, logView)
			menuPages.HidePage("search")
		})

	menuPages.AddPage("menu", menu, true, true)
	menuPages.AddPage("search", searchInput, true, false)

	logLevelDropDown.AddItem("Emergency", "Only Emergencies", 0, nil).
		AddItem("Alert", "Alerts or worse", 0, nil).
		AddItem("Critical", "Critical or worse", 0, nil).
		AddItem("Error", "Errors or worse", 0, nil).
		AddItem("Warning", "Warnings or worse", 0, nil).
		AddItem("Notice", "Notice or worse", 0, nil).
		AddItem("Informational", "Informational or worse", 0, nil).
		AddItem("Debug", "Debug or worse", 0, nil)
	logLevelDropDown.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		filter = logLevelFilter(index)
		pages.HidePage("dropdown_loglevel")
		selectLog(pipe, list.Model[list.GetCurrentItem()], filter, logView)
	})

	menu.AddItem("Active Services", tcell.KeyF1, func() {
		activeOnly = !activeOnly
		if activeOnly {
			menu.Items[0].Text = "Active Services"
		} else {
			menu.Items[0].Text = "All Services"
		}
		list.loadModel(true, activeOnly)
		app.SetFocus(list)
	})
	menu.AddItem("Log-level", tcell.KeyF2, func() {
		pages.ShowPage("dropdown_loglevel")
	})
	menu.AddItem("Filter", tcell.KeyF3, func() {
		menuPages.ShowPage("search")
		app.SetFocus(searchInput)
	})

	// Create the main layout.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(menuPages, 1, 1, false)

	selectLog(pipe, list.Model[0], filter, logView)

	return layout, nil
}

func logLevelFilter(index int) []sdjournal.Match {
	f := []sdjournal.Match{}

	for i := 0; i <= index; i++ {
		f = append(f, sdjournal.Match{
			Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
			Value: strconv.FormatInt(int64(i), 10),
		})
	}

	return f
}

func selectLog(pipe *LogPipe, log ServiceItem, filter []sdjournal.Match, logView *tview.TextView) {
	// cancel previous reader
	if pipe != nil {
		pipe.Cancel <- time.Now()
	}

	logView.Clear()

	title := log.Name
	if len(search) > 0 {
		title += fmt.Sprintf(" (filtered by %s)", search)
	}
	logView.SetTitle(title)
	logView.ScrollToEnd()

	pipe = logPipe(log.Matches, filter)
	go pipeReader(pipe, logView)

	app.SetFocus(logView)
}

func init() {
	RootCmd.AddCommand(logsCmd)
}
