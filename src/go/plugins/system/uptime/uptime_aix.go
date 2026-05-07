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

package uptime

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// getUptime returns seconds since LPAR boot.
//
// On AIX /proc/0 is the kernel scheduler entry; its mtime corresponds to
// the time the scheduler was started, i.e. boot time. This is an
// approximation accurate to the second and avoids cgo / libperfstat for
// the skeleton phase. A perfstat_partition_total based implementation can
// replace this when the perfstat plugin port lands.
func getUptime() (int, error) {
	fi, err := os.Stat("/proc/0")
	if err != nil {
		return 0, fmt.Errorf("Cannot read boot time: %s", err.Error())
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("Cannot read boot time: stat sys cast failed")
	}
	return int(time.Now().Unix() - int64(st.Mtim.Sec)), nil
}
