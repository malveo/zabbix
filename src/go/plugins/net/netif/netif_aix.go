/*
** Copyright (C) 2001-2026 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package netif

import (
	"encoding/json"
	"errors"

	"github.com/power-devops/perfstat"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

func init() {
	err := plugin.RegisterMetrics(
		&impl, "NetIf",
		"net.if.collisions", "Returns number of out-of-window collisions.",
		"net.if.in", "Returns incoming traffic statistics on network interface.",
		"net.if.out", "Returns outgoing traffic statistics on network interface.",
		"net.if.total", "Returns sum of incoming and outgoing traffic statistics on network interface.",
		"net.if.discovery", "Returns list of network interfaces. Used for low-level discovery.",
	)
	if err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

func findIface(name string) (*perfstat.NetIface, error) {
	ifs, err := perfstat.NetIfaceStat()
	if err != nil {
		return nil, err
	}
	for i := range ifs {
		if ifs[i].Name == name {
			return &ifs[i], nil
		}
	}
	return nil, errors.New("Network interface not found.")
}

func ifaceField(n *perfstat.NetIface, mode string, dir dirFlag) (uint64, error) {
	switch mode {
	case "", "bytes":
		switch dir {
		case dirIn:
			return uint64(n.IBytes), nil
		case dirOut:
			return uint64(n.OBytes), nil
		case dirIn | dirOut:
			return uint64(n.IBytes + n.OBytes), nil
		}
	case "packets":
		switch dir {
		case dirIn:
			return uint64(n.IPackets), nil
		case dirOut:
			return uint64(n.OPackets), nil
		case dirIn | dirOut:
			return uint64(n.IPackets + n.OPackets), nil
		}
	case "errors":
		switch dir {
		case dirIn:
			return uint64(n.IErrors), nil
		case dirOut:
			return uint64(n.OErrors), nil
		case dirIn | dirOut:
			return uint64(n.IErrors + n.OErrors), nil
		}
	case "dropped":
		switch dir {
		case dirIn:
			return uint64(n.IfIqDrops), nil
		case dirOut:
			return uint64(n.XmitDrops), nil
		case dirIn | dirOut:
			return uint64(n.IfIqDrops + n.XmitDrops), nil
		}
	case "collisions":
		// libperfstat exposes collisions only at the interface level.
		return uint64(n.Collisions), nil
	}
	return 0, errors.New("Unsupported metric mode on AIX.")
}

func (p *Plugin) Export(key string, params []string, ctx plugin.ContextProvider) (interface{}, error) {
	switch key {
	case "net.if.discovery":
		if len(params) > 0 {
			return nil, errors.New(errorParametersNotAllowed)
		}
		ifs, err := perfstat.NetIfaceStat()
		if err != nil {
			return nil, err
		}
		out := make([]msgIfDiscovery, 0, len(ifs))
		for i := range ifs {
			name := ifs[i].Name
			out = append(out, msgIfDiscovery{Ifname: name})
		}
		b, err := json.Marshal(out)
		if err != nil {
			return nil, err
		}
		return string(b), nil

	case "net.if.collisions":
		if len(params) < 1 || params[0] == "" {
			return nil, errors.New(errorEmptyIfName)
		}
		if len(params) > 1 {
			return nil, errors.New(errorTooManyParams)
		}
		n, err := findIface(params[0])
		if err != nil {
			return nil, err
		}
		return uint64(n.Collisions), nil
	}

	var direction dirFlag
	switch key {
	case "net.if.in":
		direction = dirIn
	case "net.if.out":
		direction = dirOut
	case "net.if.total":
		direction = dirIn | dirOut
	default:
		return nil, errors.New(errorUnsupportedMetric)
	}

	if len(params) < 1 || params[0] == "" {
		return nil, errors.New(errorEmptyIfName)
	}
	if len(params) > 2 {
		return nil, errors.New(errorTooManyParams)
	}
	mode := "bytes"
	if len(params) == 2 && params[1] != "" {
		mode = params[1]
	}

	n, err := findIface(params[0])
	if err != nil {
		return nil, err
	}
	return ifaceField(n, mode, direction)
}
