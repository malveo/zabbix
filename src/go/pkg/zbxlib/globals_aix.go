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

package zbxlib

/*
#cgo CFLAGS: -I${SRCDIR}/../../../libs/zbxsysinfo/common
#cgo LDFLAGS: ${SRCDIR}/../../../zabbix_agent/logfiles/libzbxlogfiles.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxnum/libzbxnum.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxstr/libzbxstr.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxfile/libzbxfile.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxparam/libzbxparam.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxexpr/libzbxexpr.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxip/libzbxip.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxcomms/libzbxcomms.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxbincommon/libzbxbincommon.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxcommon/libzbxcommon.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxcrypto/libzbxcrypto.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxthreads/libzbxthreads.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxmutexs/libzbxmutexs.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxnix/libzbxnix.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxhttp/libzbxhttp.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxcompress/libzbxcompress.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxregexp/libzbxregexp.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxsysinfo/libzbxagentsysinfo.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxsysinfo/common/libcommonsysinfo.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxsysinfo/simple/libsimplesysinfo.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxexec/libzbxexec.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxalgo/libzbxalgo.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxjson/libzbxjson.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxsysinfo/aix/libspechostnamesysinfo.a
#cgo LDFLAGS: ${SRCDIR}/../../../libs/zbxsysinfo/aix/libspecsysinfo.a
#cgo LDFLAGS: -L/opt/freeware/lib -L/usr/lib
#cgo LDFLAGS: -lperfstat -lpthread -lz
#cgo LDFLAGS: /opt/freeware/lib/libiconv.a
#cgo pcre2 LDFLAGS: -lpcre2-8
#cgo LDFLAGS: -Wl,-bbigtoc -Wl,-bnoquiet

#include "zbxsysinfo.h"
#include "zbxcomms.h"
#include "zbxlog.h"
#include "../src/zabbix_agent/metrics/metrics.h"
#include "../src/zabbix_agent/logfiles/logfiles.h"

typedef zbx_active_metric_t* ZBX_ACTIVE_METRIC_LP;
typedef zbx_vector_ptr_t * zbx_vector_ptr_lp_t;
typedef zbx_vector_expression_t * zbx_vector_expression_lp_t;

zbx_metric_t	parameters_agent[] = {NULL};
zbx_metric_t	parameters_specific[] = {NULL};

int	zbx_procstat_collector_started(void)
{
	return FAIL;
}

int	zbx_procstat_get_util(const char *procname, const char *username, const char *cmdline, zbx_uint64_t flags,
		int period, int type, double *value, char **errmsg)
{
	return FAIL;
}

int	get_cpustat(AGENT_RESULT *result, int cpu_num, int state, int mode)
{
	return SYSINFO_RET_FAIL;
}

char	*zbx_strerror_from_system(zbx_syserror_t error)
{
	return zbx_strerror(errno);
}

// AIX vmstat collector bootstrap.
// Classic agentd spawns a dedicated thread that calls collect_vmstat_data
// every second; agent2 has no such thread so system_stat() in
// libspecsysinfo.a sees a NULL collector and returns
// "Collector is not started." for every system.stat[*] key.
// We replicate that loop here.
#include <pthread.h>
#include <unistd.h>
#include "stats.h"

extern void collect_vmstat_data(ZBX_VMSTAT_DATA *vmstat);

static void *zbxaix_vmstat_loop(void *arg)
{
	zbx_collector_data	*c;

	(void)arg;
	while (1)
	{
		c = get_collector();
		if (NULL != c)
		{
			// enabled flag is what system_stat() flips on first call;
			// collect unconditionally so data is fresh on first read.
			c->vmstat.enabled = 1;
			collect_vmstat_data(&c->vmstat);
			c->vmstat.data_available = 1;
		}
		sleep(1);
	}
	return NULL;
}

int	zbxaix_start_collector(void)
{
	char			*err = NULL;
	pthread_t		t;
	pthread_attr_t		attr;
	zbx_collector_data	*c;

	if (SUCCEED != zbx_init_collector_data(&err))
	{
		if (NULL != err) zbx_free(err);
		return -1;
	}
	// update_vmstat() in vmstats.c only saves a baseline on the first
	// call and emits deltas on the second. Prime it twice with a 1s
	// gap so test-mode (single -t invocation) sees non-zero values.
	c = get_collector();
	if (NULL != c)
	{
		c->vmstat.enabled = 1;
		collect_vmstat_data(&c->vmstat);
		sleep(1);
		collect_vmstat_data(&c->vmstat);
		c->vmstat.data_available = 1;
	}
	pthread_attr_init(&attr);
	pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
	if (0 != pthread_create(&t, &attr, zbxaix_vmstat_loop, NULL))
	{
		pthread_attr_destroy(&attr);
		return -2;
	}
	pthread_attr_destroy(&attr);
	return 0;
}

*/
import "C"

// StartAIXCollector boots the libperfstat-based vmstat collector that
// system.stat[*] keys read from. Returns 0 on success, negative on
// failure (typically shared-memory exhaustion or pthread refusal).
// Safe to call once at agent startup; idempotent calls would leak.
func StartAIXCollector() int {
	return int(C.zbxaix_start_collector())
}
