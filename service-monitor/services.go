package main

import (
	"strings"

	"github.com/coreos/go-systemd/dbus"
)

type Services []dbus.UnitStatus

func services() (Services, error) {
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
		if !strings.HasSuffix(v.Name, ".service") {
			continue
		}

		res = append(res, v)
	}

	return res, nil
}

func (ts Services) Contains(name string) bool {
	for _, t := range ts {
		if t.Name == name {
			return true
		}
	}

	return false
}

func (ts Services) Strings() []string {
	var res []string
	for _, t := range ts {
		res = append(res, t.Name)
	}

	return res
}
