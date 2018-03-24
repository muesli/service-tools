package main

import (
	"fmt"
	"sort"

	"github.com/coreos/go-systemd/sdjournal"
)

type ServiceItem struct {
	Name        string
	Description string
	Matches     []sdjournal.Match
}

func serviceModel(activeOnly bool) ([]ServiceItem, error) {
	ts, err := services()
	if err != nil {
		return []ServiceItem{}, fmt.Errorf("Can't find systemd services: %s", err)
	}
	if activeOnly {
		ts = ts.ActiveOnly()
	}

	sort.Slice(ts, func(i, j int) bool {
		return ts[i].Name < ts[j].Name
	})

	model := []ServiceItem{
		{
			Name:        "All",
			Description: "Everything in the log",
			Matches:     []sdjournal.Match{},
		},
		{
			Name:        "Kernel",
			Description: "Kernel log",
			Matches: []sdjournal.Match{
				{
					Field: sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER,
					Value: "kernel",
				},
			},
		},
	}
	for _, service := range ts {
		model = append(model, ServiceItem{
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

	return model, nil
}
