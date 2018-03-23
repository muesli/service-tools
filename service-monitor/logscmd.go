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

	mainReader *LogPipe
	logView    *tview.TextView
	info       *tview.TextView
	model      []ServiceItem
	filter     = logLevelFilter(6)
	activeOnly = true
	search     string
)

func logsForm() (tview.Primitive, error) {
	list := tview.NewList()
	logView = tview.NewTextView()
	errLogView := tview.NewTextView()
	errLogView.ScrollToEnd()

	list.
		SetBorder(true).
		SetTitle("Logs")
	list.
		SetDoneFunc(func() {
			app.SetFocus(logView)
		})

	err := fillLogModel(list, activeOnly)
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

	errReader := logPipe([]sdjournal.Match{
		{
			Field: sdjournal.SD_JOURNAL_FIELD_PRIORITY,
			Value: "3",
		},
	})
	go pipeReader(errReader, errLogView)

	list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		selectLog(model[index])
	})

	flex := tview.NewFlex().
		AddItem(list, 32, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(logView, 0, 8, false).
			AddItem(errLogView, 8, 1, false), 0, 1, false)

	info = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(false)
	updateMenu()

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
			selectLog(model[list.GetCurrentItem()])
			updateMenu()
			menuPages.HidePage("search")
		})

	menuPages.AddPage("menu", info, true, true)
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
		selectLog(model[list.GetCurrentItem()])
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			activeOnly = !activeOnly
			fillLogModel(list, activeOnly)
			updateMenu()
			app.SetFocus(list)
		case tcell.KeyF2:
			pages.ShowPage("dropdown_loglevel")
		case tcell.KeyF3:
			menuPages.ShowPage("search")
			app.SetFocus(searchInput)
		}

		return event
	})

	// Create the main layout.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(menuPages, 1, 1, false)

	selectLog(model[0])

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

func fillLogModel(list *tview.List, activeOnly bool) error {
	var err error
	list.Clear()

	if activeOnly {
		list.SetTitle("Active Services")
	} else {
		list.SetTitle("All Services")
	}

	model, err = logsModel(activeOnly)
	if err != nil {
		return err
	}

	for _, srv := range model {
		list.AddItem(srv.Name, srv.Description, 0, nil)
	}

	return nil
}

func updateMenu() {
	info.Clear()
	if activeOnly {
		fmt.Fprintf(info, `%s ["%d"][darkcyan]%s[white][""]  `, "F1", 0, "All Services")
	} else {
		fmt.Fprintf(info, `%s ["%d"][darkcyan]%s[white][""]  `, "F1", 0, "Active Services")
	}

	fmt.Fprintf(info, `%s ["%d"][darkcyan]%s[white][""]  `, "F2", 1, "Log-level")
	fmt.Fprintf(info, `%s ["%d"][darkcyan]%s[white][""]  `, "F3", 2, "Filter")
}

func selectLog(log ServiceItem) {
	// cancel previous reader
	if mainReader != nil {
		mainReader.Cancel <- time.Now()
	}

	logView.Clear()

	title := log.Name
	if len(search) > 0 {
		title += fmt.Sprintf(" (filtered by %s)", search)
	}
	logView.SetTitle(title)
	logView.ScrollToEnd()

	mainReader = logPipe(log.Matches)
	go pipeReader(mainReader, logView)

	app.SetFocus(logView)
}

func init() {
	RootCmd.AddCommand(logsCmd)
}
