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

package kernel

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.zabbix.com/sdk/zbxerr"
)

// getFirstNum returns the kernel-side numeric limits exposed via lsattr.
// AIX has no /proc/sys equivalent. Mappings:
//   - kernel.maxproc  -> lsattr -El sys0 -a maxuproc (per-user proc limit)
//   - kernel.maxfiles -> RLIMIT_NOFILE rlim_max (process FD limit)
//   - kernel.openfiles -> not exposed without scanning /proc/<pid>/file_count;
//     return UnsupportedMetric for now (Phase 4 perfstat may help).
func getFirstNum(key string) (uint64, error) {
	switch key {
	case "kernel.maxproc":
		out, err := exec.Command("/usr/sbin/lsattr", "-El", "sys0", "-a", "maxuproc",
			"-F", "value").Output()
		if err != nil {
			return 0, fmt.Errorf("lsattr maxuproc failed: %s", err)
		}
		return strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)

	case "kernel.maxfiles":
		var rlim syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
			return 0, fmt.Errorf("getrlimit RLIMIT_NOFILE: %s", err)
		}
		return uint64(rlim.Max), nil

	case "kernel.openfiles":
		return 0, zbxerr.ErrorUnsupportedMetric
	}

	return 0, zbxerr.ErrorUnsupportedMetric
}
