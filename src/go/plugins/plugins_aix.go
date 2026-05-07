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
	_ "golang.zabbix.com/agent2/plugins/log"
	_ "golang.zabbix.com/agent2/plugins/net/dns"
	_ "golang.zabbix.com/agent2/plugins/net/tcp"
	// system/swap omitted on AIX: swap_nix.go uses syscall.Sysinfo which is Linux-only.
	// Phase 3 (perfstat plugin port) replaces it.
	_ "golang.zabbix.com/agent2/plugins/system/uname"
	_ "golang.zabbix.com/agent2/plugins/system/uptime"
	_ "golang.zabbix.com/agent2/plugins/system/users"
	_ "golang.zabbix.com/agent2/plugins/systemrun"
	_ "golang.zabbix.com/agent2/plugins/vfs/dir"
	_ "golang.zabbix.com/agent2/plugins/vfs/file"
	_ "golang.zabbix.com/agent2/plugins/vfs/fs"
	_ "golang.zabbix.com/agent2/plugins/web/certificate"
	_ "golang.zabbix.com/agent2/plugins/web/page"
	_ "golang.zabbix.com/agent2/plugins/zabbix/async"
	_ "golang.zabbix.com/agent2/plugins/zabbix/stats"
	_ "golang.zabbix.com/agent2/plugins/zabbix/sync"
)
