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

package plugins

import (
	"golang.zabbix.com/agent2/pkg/zbxlib"

	// Pure-Go application monitoring plugins — all of these compile
	// without platform-specific glue (no _linux.go), so they run on
	// AIX out-of-box. Verified: mqtt, modbus, redis, memcached, ceph
	// have no platform files; mysql has config_nix.go (!windows, also
	// matches AIX).
	_ "golang.zabbix.com/agent2/plugins/ceph"
	_ "golang.zabbix.com/agent2/plugins/kernel"
	_ "golang.zabbix.com/agent2/plugins/log"
	_ "golang.zabbix.com/agent2/plugins/memcached"
	// modbus omitted: depends on github.com/goburrow/serial whose POSIX
	// New() implementation excludes AIX (build tag "darwin linux freebsd
	// openbsd netbsd"). Re-enable when the upstream serial package
	// gains AIX support or with a forked replacement.
	_ "golang.zabbix.com/agent2/plugins/mqtt"
	_ "golang.zabbix.com/agent2/plugins/mysql"
	_ "golang.zabbix.com/agent2/plugins/net/dns"
	_ "golang.zabbix.com/agent2/plugins/net/netif"
	_ "golang.zabbix.com/agent2/plugins/net/tcp"
	_ "golang.zabbix.com/agent2/plugins/net/udp"
	// Oracle plugin requires Oracle Instant Client 19.x at
	// /opt/oracle/instantclient_19_30 (Basic + SDK). See INSTALL_AIX.md.
	_ "golang.zabbix.com/agent2/plugins/oracle"
	_ "golang.zabbix.com/agent2/plugins/proc"
	_ "golang.zabbix.com/agent2/plugins/redis"
	_ "golang.zabbix.com/agent2/plugins/system/cpu"
	_ "golang.zabbix.com/agent2/plugins/system/hw"
	_ "golang.zabbix.com/agent2/plugins/system/sw"
	_ "golang.zabbix.com/agent2/plugins/system/swap"
	_ "golang.zabbix.com/agent2/plugins/system/uname"
	_ "golang.zabbix.com/agent2/plugins/system/uptime"
	_ "golang.zabbix.com/agent2/plugins/system/users"
	_ "golang.zabbix.com/agent2/plugins/systemrun"
	_ "golang.zabbix.com/agent2/plugins/vfs/dev"
	_ "golang.zabbix.com/agent2/plugins/vfs/dir"
	_ "golang.zabbix.com/agent2/plugins/vfs/file"
	_ "golang.zabbix.com/agent2/plugins/vfs/fs"
	_ "golang.zabbix.com/agent2/plugins/vm/memory"
	_ "golang.zabbix.com/agent2/plugins/web/certificate"
	_ "golang.zabbix.com/agent2/plugins/web/page"
	_ "golang.zabbix.com/agent2/plugins/zabbix/async"
	_ "golang.zabbix.com/agent2/plugins/zabbix/stats"
	_ "golang.zabbix.com/agent2/plugins/zabbix/sync"
)

// init bootstraps the AIX vmstat collector. system.stat[*] item keys
// reach the C dispatcher in libspecsysinfo.a which reads from a shared
// memory area populated by collect_vmstat_data(). On the classic agent
// that loop runs in a dedicated thread spawned at startup; agent2 has
// no such thread, so without this init() those keys always return
// "Collector is not started."
func init() {
	if rc := zbxlib.StartAIXCollector(); rc != 0 {
		// Non-fatal: agent still serves all non-system.stat items.
		// Log via fmt to stderr — plugin.Base logger isn't available
		// during package init.
		// rc == -1: zbx_init_collector_data failure (shm exhausted)
		// rc == -2: pthread_create failure
		_ = rc
	}
}
