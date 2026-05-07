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

package vfsdev

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/power-devops/perfstat"
)

// devRecord mirrors the JSON envelope used by the Linux backend.
type devRecord struct {
	Name string `json:"{#DEVNAME}"`
	Type string `json:"{#DEVTYPE}"`
}

// AIX libperfstat reports per-disk Rblks/Wblks (in 512-byte blocks) and
// a single Xfers counter for combined transfers — there is no separate
// read/write transfer split. To keep the vfs.dev item semantics stable
// we approximate by attributing half of Xfers to each direction.

func (p *Plugin) getDiscovery() (string, error) {
	disks, err := perfstat.DiskStat()
	if err != nil {
		return "", err
	}
	out := make([]*devRecord, 0, len(disks))
	for i := range disks {
		out = append(out, &devRecord{Name: disks[i].Name, Type: "disk"})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Plugin) getDeviceName(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	// AIX disk names are simple identifiers like "hdisk0"; strip any
	// /dev/ prefix the user may have typed.
	name = strings.TrimPrefix(name, "/dev/")
	return name, nil
}

func diskToStats(d *perfstat.Disk) *devStats {
	half := uint64(d.Xfers / 2)
	return &devStats{
		rx: devIO{
			sectors:    uint64(d.Rblks),
			operations: half,
		},
		tx: devIO{
			sectors:    uint64(d.Wblks),
			operations: uint64(d.Xfers) - half,
		},
	}
}

func (p *Plugin) getDeviceStats(name string) (*devStats, error) {
	disks, err := perfstat.DiskStat()
	if err != nil {
		return nil, err
	}
	name = strings.TrimPrefix(name, "/dev/")

	if name == "" {
		// Aggregate across all disks.
		var agg devStats
		for i := range disks {
			s := diskToStats(&disks[i])
			agg.rx.sectors += s.rx.sectors
			agg.rx.operations += s.rx.operations
			agg.tx.sectors += s.tx.sectors
			agg.tx.operations += s.tx.operations
		}
		return &agg, nil
	}

	for i := range disks {
		if disks[i].Name == name {
			return diskToStats(&disks[i]), nil
		}
	}
	return nil, errors.New("Disk not found.")
}

func (p *Plugin) collectDeviceStats(devices map[string]*devUnit) error {
	now := time.Now()
	for _, dev := range devices {
		stats, err := p.getDeviceStats(dev.name)
		if err != nil || stats == nil {
			continue
		}
		stats.clock = now.UnixNano()
		dev.history[dev.tail] = *stats
		if dev.tail = dev.tail.inc(); dev.tail == dev.head {
			dev.head = dev.head.inc()
		}
	}
	return nil
}
