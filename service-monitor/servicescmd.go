package main

import (
	"fmt"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

var (
	servicesCmd = &cobra.Command{
		Use:   "services",
		Short: "monitor service states",
		Long:  `The services command starts an interactive service monitor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			form, err := servicesForm()
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

func servicesForm() (tview.Primitive, error) {
	var pipe *LogPipe
	activeOnly := false
	filter := logLevelFilter(6)

	pages := tview.NewPages()
	confirmDialog := tview.NewModal()
	list := NewServicesView()
	list.SetMainTextColor(tcell.GetColor(theme.Colors["Service"]))
	list.SetSecondaryTextColor(tcell.GetColor(theme.Colors["Description"]))

	err := list.loadModel(false, activeOnly)
	if err != nil {
		return nil, err
	}

	infoTable := tview.NewTable().
		SetBorders(false)
	infoTable.SetCell(0, 0, tview.NewTableCell("State:"))
	infoTable.SetCell(1, 0, tview.NewTableCell("Description:"))
	infoTable.SetCell(3, 0, tview.NewTableCell("Load Successful:"))
	infoTable.SetCell(4, 0, tview.NewTableCell("SubState:"))

	serviceView := tview.NewFlex().
		SetDirection(tview.FlexRow)
	serviceView.
		SetBorder(true).
		SetTitle("Service")

	logView := tview.NewTextView()
	list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		app.SetFocus(logView)
	})
	list.SetChangedFunc(func(index int, primText, secText string, shortcut rune) {
		pipe = selectService(pipe, list.Model[index], filter, logView, serviceView, infoTable, confirmDialog)
	})

	logView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(list)
		})
	logView.
		SetBorder(true).
		SetTitle("Log")

	serviceView.
		AddItem(infoTable, 6, 1, false).
		AddItem(logView, 0, 1, false)

	flex := tview.NewFlex().
		AddItem(list, 40, 1, true).
		AddItem(serviceView, 0, 1, false)

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

	confirmDialog.
		SetText("Do you want to start the service?").
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.HidePage("confirm_dialog")
			if buttonIndex == 0 {
				err := toggleServiceState(list)
				if err != nil {
					panic(err)
				}
				updateServiceUI(list.Model[list.GetCurrentItem()].Name, serviceView, infoTable, confirmDialog)
			}
			app.SetFocus(list)
		})

	pages.AddPage("flex", flex, true, true)
	pages.AddPage("dropdown_loglevel", logLevelDialog, true, false)
	pages.AddPage("confirm_dialog", confirmDialog, false, false)

	menuPages := tview.NewPages()
	searchInput := tview.NewInputField()
	searchInput.
		SetLabel("Search for: ").
		SetFieldWidth(40).
		SetAcceptanceFunc(nil).
		SetDoneFunc(func(key tcell.Key) {
			search = searchInput.GetText()
			menuPages.HidePage("search")
			pipe = selectService(pipe, list.Model[list.GetCurrentItem()], filter, logView, serviceView, infoTable, confirmDialog)
			app.SetFocus(list)
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
		pipe = selectService(pipe, list.Model[list.GetCurrentItem()], filter, logView, serviceView, infoTable, confirmDialog)
	})

	menu.AddItem("Active Services", tcell.KeyF1, func() {
		activeOnly = !activeOnly
		if activeOnly {
			menu.Items[0].Text = "All Services"
		} else {
			menu.Items[0].Text = "Active Services"
		}
		list.loadModel(false, activeOnly)
		app.SetFocus(list)
	})
	menu.AddItem("Log-level", tcell.KeyF2, func() {
		pages.ShowPage("dropdown_loglevel")
	})
	menu.AddItem("Filter", tcell.KeyF3, func() {
		menuPages.ShowPage("search")
		app.SetFocus(searchInput)
	})
	menu.AddItem("Start Service", tcell.KeyF8, func() {
		pages.ShowPage("confirm_dialog")
	})

	// Create the main layout.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(menuPages, 1, 1, false)

	pipe = selectService(pipe, list.Model[0], filter, logView, serviceView, infoTable, confirmDialog)

	return layout, nil
}

func updateServiceUI(name string, serviceView *tview.Flex, infoTable *tview.Table, confirmDialog *tview.Modal) {
	u, err := service(name)
	if err != nil {
		panic(err)
	}

	if u.ActiveState == "active" {
		menu.Items[3].Text = "Stop Service"
		confirmDialog.SetText(fmt.Sprintf("Do you want to stop service %s?", u.Name))
	} else {
		menu.Items[3].Text = "Start Service"
		confirmDialog.SetText(fmt.Sprintf("Do you want to start service %s?", u.Name))
	}

	serviceView.SetTitle(u.Name)
	infoTable.SetCell(0, 1, tview.NewTableCell(u.ActiveState))
	infoTable.SetCell(1, 1, tview.NewTableCell(u.Description))
	infoTable.SetCell(3, 1, tview.NewTableCell(u.LoadState))
	infoTable.SetCell(4, 1, tview.NewTableCell(u.SubState))
}

func selectService(pipe *LogPipe, l ServiceItem, filter []sdjournal.Match, logView *tview.TextView, serviceView *tview.Flex, infoTable *tview.Table, confirmDialog *tview.Modal) *LogPipe {
	updateServiceUI(l.Name, serviceView, infoTable, confirmDialog)

	// cancel previous reader
	if pipe != nil {
		pipe.Cancel <- time.Now()
		pipe.WaitGroup.Wait()
	}

	logView.Clear()

	title := "Log"
	if len(search) > 0 {
		title += fmt.Sprintf(" (filtered by %s)", search)
	}
	logView.SetTitle(title)
	logView.ScrollToEnd()

	pipe = logPipe(l.Matches, filter)
	go pipeReader(pipe, logView)

	return pipe
}

func toggleServiceState(list *ServicesView) error {
	s := list.Model[list.GetCurrentItem()]
	u, err := service(s.Name)
	if err != nil {
		return err
	}

	if u.ActiveState == "active" {
		err = stopService(s.Name)
	} else {
		err = startService(s.Name)
	}

	return err
}

func init() {
	RootCmd.AddCommand(servicesCmd)
}
