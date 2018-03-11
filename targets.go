package main

import (
	"strings"

	"github.com/coreos/go-systemd/dbus"
)

type Targets []dbus.UnitStatus

func targets() (Targets, error) {
	res := []dbus.UnitStatus{}
	conn, err := dbus.New()
	if err != nil {
		return res, err
	}
	us, err := conn.ListUnits()
	if err != nil {
		return res, err
	}
	for _, v := range us {
		if !strings.HasSuffix(v.Name, ".target") {
			continue
		}

		res = append(res, v)
	}

	return res, nil
}

func (ts Targets) Contains(name string) bool {
	for _, t := range ts {
		if t.Name == name {
			return true
		}
	}

	return false
}
