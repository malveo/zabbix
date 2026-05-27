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

package hw

import (
	"os/exec"
	"strings"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

// Plugin -
type Plugin struct {
	plugin.Base
}

var impl Plugin

func init() {
	err := plugin.RegisterMetrics(
		&impl, "Hw",
		"system.hw.chassis", "Chassis information.",
		"system.hw.devices", "Listing of installed devices.",
	)
	if err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

// Configure -
func (p *Plugin) Configure(global *plugin.GlobalOptions, options interface{}) {}

// Validate -
func (p *Plugin) Validate(options interface{}) error { return nil }

// Export -
func (p *Plugin) Export(key string, params []string, ctx plugin.ContextProvider) (interface{}, error) {
	switch key {
	case "system.hw.chassis":
		return aixChassis()
	case "system.hw.devices":
		return aixDevices()
	default:
		return nil, plugin.UnsupportedMetricError
	}
}

// aixChassis returns vendor + model + serial harvested from `prtconf`.
// AIX has no SMBIOS tables; the canonical vendor info source is prtconf,
// which prints System Model, Machine Serial Number, Processor Type/Impl
// version. We extract only the fields visible from a non-root user.
func aixChassis() (string, error) {
	out, err := exec.Command("/usr/sbin/prtconf").Output()
	if err != nil {
		return "", err
	}

	var model, serial, mfg string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "System Model:"):
			model = strings.TrimSpace(strings.TrimPrefix(line, "System Model:"))
		case strings.HasPrefix(line, "Machine Serial Number:"):
			serial = strings.TrimSpace(strings.TrimPrefix(line, "Machine Serial Number:"))
		case strings.HasPrefix(line, "Processor Type:"):
			// not used as vendor but kept for completeness; vendor is implied
			// by IBM AIX and prtconf does not expose a "Manufacturer:" line
			// in non-root mode.
		}
	}
	if mfg == "" {
		mfg = "IBM"
	}
	parts := []string{}
	for _, p := range []string{mfg, model, serial} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "", errs.New("cannot obtain hardware information")
	}
	return strings.Join(parts, " "), nil
}

// aixDevices returns the output of `lscfg -v` (verbose configuration
// listing) — analogue of lspci/lsusb on Linux but covering AIX-specific
// virtual and physical adapters (vSCSI, FC, ent, etc.).
func aixDevices() (string, error) {
	out, err := exec.Command("/usr/sbin/lscfg", "-v").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
