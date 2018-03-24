package main

import "github.com/rivo/tview"

type ServicesView struct {
	*tview.List

	Model []ServiceItem
}

func NewServicesView() *ServicesView {
	v := &ServicesView{
		List: tview.NewList(),
	}

	v.
		SetBorder(true).
		SetTitle("Services")
	v.
		SetDoneFunc(func() {
			// app.SetFocus(logView)
		})

	return v
}

func (list *ServicesView) loadModel(specialServices, activeOnly bool) error {
	var err error
	list.Clear()

	if activeOnly {
		list.SetTitle("Active Services")
	} else {
		list.SetTitle("All Services")
	}

	list.Model, err = serviceModel(specialServices, activeOnly)
	if err != nil {
		return err
	}

	for _, srv := range list.Model {
		list.AddItem(srv.Name, srv.Description, 0, nil)
	}

	return nil
}
