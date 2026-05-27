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
	"os/exec"
	"strings"
	"time"
)

// getUptime returns seconds since LPAR boot.
//
// AIX has no /proc/stat with btime field. We parse `who -b` output:
//
//	   .        system boot May 07 15:57
//
// The year is implicit; assume current year (close enough for sub-year
// uptimes which are the common case). A perfstat-based variant lives in
// Phase 4 and replaces this once libperfstat bindings land.
func getUptime() (int, error) {
	out, err := exec.Command("/usr/bin/who", "-b").Output()
	if err != nil {
		return 0, fmt.Errorf("Cannot read boot time: %s", err)
	}

	idx := strings.Index(string(out), "system boot")
	if idx < 0 {
		return 0, fmt.Errorf("Cannot read boot time: unexpected who -b output: %q", string(out))
	}
	rest := strings.TrimSpace(string(out)[idx+len("system boot"):])

	fields := strings.Fields(rest)
	if len(fields) < 3 {
		return 0, fmt.Errorf("Cannot read boot time: short who -b output: %q", rest)
	}
	stamp := strings.Join(fields[:3], " ")

	now := time.Now()
	loc := now.Location()

	t, err := time.ParseInLocation("Jan 02 15:04", stamp, loc)
	if err != nil {
		return 0, fmt.Errorf("Cannot parse boot time %q: %s", stamp, err)
	}
	t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	if t.After(now) {
		t = t.AddDate(-1, 0, 0)
	}
	return int(now.Unix() - t.Unix()), nil
}
