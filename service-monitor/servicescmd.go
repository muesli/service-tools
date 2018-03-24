package main

import (
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
	activeOnly := false

	list := NewServicesView()
	err := list.loadModel(activeOnly)
	if err != nil {
		return nil, err
	}

	serviceView := tview.NewTextView()
	serviceView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetDoneFunc(func(key tcell.Key) {
			var next tview.Primitive = list
			app.SetFocus(next)
		})
	serviceView.
		SetBorder(true).
		SetTitle("Service")

	list.SetSelectedFunc(func(index int, primText, secText string, shortcut rune) {
		// selectLog(list.Model[index])
	})

	flex := tview.NewFlex().
		AddItem(list, 0, 1, true).
		AddItem(serviceView, 40, 1, false)

	pages := tview.NewPages()
	pages.AddPage("flex", flex, true, true)
	// pages.AddPage("dropdown_loglevel", logLevelDialog, true, false)

	menuPages := tview.NewPages()
	searchInput := tview.NewInputField()
	menuPages.AddPage("menu", menu, true, true)
	menuPages.AddPage("search", searchInput, true, false)

	searchInput.
		SetLabel("Search for: ").
		SetFieldWidth(40).
		SetAcceptanceFunc(nil).
		SetDoneFunc(func(key tcell.Key) {
			search = searchInput.GetText()
			menuPages.HidePage("search")
		})

	menu.AddItem("Active Services", tcell.KeyF1, func() {
		activeOnly = !activeOnly
		if activeOnly {
			menu.Items[0].Text = "Active Services"
		} else {
			menu.Items[0].Text = "All Services"
		}
		list.loadModel(activeOnly)
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
	})

	// Create the main layout.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(menuPages, 1, 1, false)

	return layout, nil
}

func init() {
	RootCmd.AddCommand(servicesCmd)
}
