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

package memory

import (
	"errors"

	"github.com/power-devops/perfstat"
	"golang.zabbix.com/sdk/zbxerr"
)

// AIX libperfstat reports memory in 4 KiB page units. We multiply by
// pageSize to expose bytes — same convention vm.memory.size returns on
// other platforms. NumPerm is the file-cache page count (closest
// analogue to Linux "Cached"); buffers has no AIX equivalent.
const pageSize = 4096

func (p *Plugin) exportVMMemorySize(mode string) (interface{}, error) {
	mem, err := perfstat.MemoryTotalStat()
	if err != nil {
		return nil, zbxerr.ErrorCannotFetchData.Wrap(err)
	}

	total := uint64(mem.RealTotal) * pageSize
	free := uint64(mem.RealFree) * pageSize
	used := uint64(mem.RealInUse) * pageSize
	cached := uint64(mem.NumPerm) * pageSize
	avail := uint64(mem.RealAvailable) * pageSize
	if avail == 0 {
		avail = free + cached
	}

	switch mode {
	case "total", "":
		return total, nil
	case "free":
		return free, nil
	case "used":
		return used, nil
	case "cached":
		return cached, nil
	case "available":
		return avail, nil
	case "buffers":
		// Linux distinguishes Buffers from Cached; AIX does not.
		return uint64(0), nil
	case "pused":
		if total == 0 {
			return 0.0, nil
		}
		return float64(used) / float64(total) * 100, nil
	case "pavailable":
		if total == 0 {
			return 0.0, nil
		}
		return float64(avail) / float64(total) * 100, nil
	case "active", "anon", "inactive", "slab":
		// Linux-specific Counters; not directly exposed by libperfstat.
		return nil, errors.New("Mode not supported on AIX.")
	}

	return nil, errors.New("Invalid first parameter.")
}
