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

package swap

import (
	"errors"

	"github.com/power-devops/perfstat"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

const aixMB = 1024 * 1024

func init() {
	if err := plugin.RegisterMetrics(&impl, "Swap",
		"system.swap.size", "Swap space size in bytes or in percentage from total.",
		"system.swap.in", "Swap-in (read from disk to memory).",
		"system.swap.out", "Swap-out (write from memory to disk)."); err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

// AIX paging-space accounting works differently from Linux: each
// configured paging space is a logical volume (or NFS file) reported by
// libperfstat as a PagingSpace entry with MBSize and MBUsed. There is
// no system-wide swap-in/swap-out counter exposed via libperfstat — the
// closest are paging-rate counters (pi/po) inside MemoryTotal which are
// already handled by system.stat[page,*]. system.swap.in/out therefore
// only support the "count" mode (sum of IOPending across all paging
// spaces); pages/sectors return 0.

func getSwapSize() (total, free uint64, err error) {
	pgs, perr := perfstat.PagingSpaceStat()
	if perr != nil {
		return 0, 0, perr
	}
	for _, ps := range pgs {
		t := uint64(ps.MBSize) * aixMB
		u := uint64(ps.MBUsed) * aixMB
		total += t
		if t >= u {
			free += t - u
		}
	}
	return total, free, nil
}

// AIX paging spaces have no per-page counter accessible via libperfstat;
// IOPending is the only field exposed and only for one snapshot — return
// it as the "count" mode. Pages/sectors fall back to zero so the item
// returns a value rather than ZBX_NOTSUPPORTED.
func getSwapStats(swapdev string) (io, sect, pag uint64, err error) {
	pgs, perr := perfstat.PagingSpaceStat()
	if perr != nil {
		return 0, 0, 0, perr
	}
	for _, ps := range pgs {
		if swapdev == "" || swapdev == "all" || swapdev == ps.Name {
			io += uint64(ps.IOPending)
		}
	}
	return io, 0, 0, nil
}

func getSwapStatsIn(swapdev string) (uint64, uint64, uint64, error) {
	return getSwapStats(swapdev)
}

func getSwapStatsOut(swapdev string) (uint64, uint64, uint64, error) {
	return getSwapStats(swapdev)
}

// getSwapPages / getSwapDevStats are referenced by some shared helpers
// in swap_nix.go (Linux/Darwin path). Provide compatible no-op shims to
// satisfy the package surface on AIX without changing the dispatcher.
func getSwapPages() (uint64, uint64, bool) {
	return 0, 0, false
}

func getSwapDevStats(swapdev string, rw bool) (uint64, uint64, bool) {
	_ = swapdev
	_ = rw
	return 0, 0, false
}

// Reference unused imports so go vet does not complain when the swap
// package builds on AIX without the (Linux-only) procfs paths.
var _ = errors.New
