package main

import (
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/dbus"
)

type Services []dbus.UnitStatus

var (
	conn *dbus.Conn
)

func services() (Services, error) {
	res := []dbus.UnitStatus{}

	var err error
	if conn == nil {
		conn, err = dbus.New()
		if err != nil {
			return res, err
		}
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

func service(name string) (dbus.UnitStatus, error) {
	var err error
	if conn == nil {
		conn, err = dbus.New()
		if err != nil {
			return dbus.UnitStatus{}, err
		}
	}

	us, err := conn.ListUnits()
	if err != nil {
		return dbus.UnitStatus{}, err
	}
	for _, v := range us {
		if v.Name == name {
			return v, nil
		}
	}

	return dbus.UnitStatus{}, fmt.Errorf("no such service: %s", name)
}

func (ts Services) ActiveOnly() Services {
	res := Services{}
	for _, t := range ts {
		if t.ActiveState != "active" {
			continue
		}

		res = append(res, t)
	}

	return res
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
