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

package zabbixasync

// AIX exposes additional system.* metric keys not present on other
// Unix platforms — primarily system.stat[*] (vmstat-style counters)
// served by the C-bridge libspecsysinfo.a. The official "AIX by
// Zabbix agent" template is built around them. Drop the Linux-only
// "sensor" key (no hwmon on AIX) and add system.stat.
//
// All metric keys here resolve through ExecuteCheck → resolveMetric
// in pkg/zbxlib/checks_aix.go, which dispatches the cgo call.
func getMetrics() []string {
	return []string{
		"system.localtime", "Returns system local time.",
		"system.boottime", "Returns system boot time.",
		"net.tcp.listen", "Checks if this TCP port is in LISTEN state.",
		// net.udp.listen disabled on AIX — the C bridge implementation
		// in libspecsysinfo.a opens a raw socket which requires root
		// privileges; running unprivileged it triggers a crash that
		// resets the agent process. Re-enable once a non-raw probe is
		// available, or when running the agent as root.
		"system.cpu.load", "CPU load.",
		"system.cpu.switches", "Count of context switches.",
		"system.cpu.intr", "Device interrupts.",
		"system.stat", "Virtual memory statistics (vmstat).",
	}
}
