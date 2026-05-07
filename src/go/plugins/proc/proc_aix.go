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

package proc

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/power-devops/perfstat"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

// AIX proc backend — minimal viable subset.
//
// Implemented:
//
//	proc.num  — process count, with optional name/user filter
//	proc.mem  — sum of real memory (KB → bytes), name/user filter
//	proc.get  — JSON snapshot via perfstat.ProcessStat()
//
// Not yet implemented (return UnsupportedMetric):
//
//	proc.cpu.util — needs scanid history + sample interval logic from
//	  proc_linux.go's cpuUtilStats. Out of scope for the skeleton port.
//
// libperfstat does not expose process commandline, so the cmdline filter
// parameter is silently ignored on AIX. Use the name parameter instead.

// Plugin -
type Plugin struct {
	plugin.Base
}

// PluginExport -
type PluginExport struct {
	plugin.Base
}

var impl Plugin
var implExport PluginExport

func init() {
	if err := plugin.RegisterMetrics(&impl, "Proc",
		"proc.cpu.util", "Process CPU utilization percentage."); err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
	if err := plugin.RegisterMetrics(&implExport, "ProcExporter",
		"proc.mem", "Process memory utilization values.",
		"proc.num", "The number of processes.",
		"proc.get", "List of OS processes with statistics."); err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

// procFilter holds the optional name and user matchers from item params.
type procFilter struct {
	name string
	user string // not yet used; perfstat returns numeric UID only
}

func parseFilter(params []string) procFilter {
	var f procFilter
	if len(params) >= 1 {
		f.name = params[0]
	}
	if len(params) >= 2 {
		f.user = params[1]
	}
	// params[2] (cmdline) ignored on AIX.
	return f
}

func (f procFilter) matches(p *perfstat.Process) bool {
	if f.name != "" && p.ProcessName != f.name {
		return false
	}
	// User filter would require name → uid resolution (getpwnam).
	// Skipped in MVP; user-filtered queries on AIX always return all
	// matches by name only.
	return true
}

// Export — proc.cpu.util (placeholder).
func (p *Plugin) Export(key string, params []string, ctx plugin.ContextProvider) (interface{}, error) {
	if key == "proc.cpu.util" {
		return nil, plugin.UnsupportedMetricError
	}
	return nil, plugin.UnsupportedMetricError
}

// Export — proc.num / proc.mem / proc.get.
func (p *PluginExport) Export(key string, params []string, ctx plugin.ContextProvider) (interface{}, error) {
	switch key {
	case "proc.num":
		return procNum(parseFilter(params))
	case "proc.mem":
		return procMem(parseFilter(params), params)
	case "proc.get":
		return procGet(parseFilter(params))
	}
	return nil, plugin.UnsupportedMetricError
}

func procNum(f procFilter) (uint64, error) {
	procs, err := perfstat.ProcessStat()
	if err != nil {
		return 0, err
	}
	var n uint64
	for i := range procs {
		if f.matches(&procs[i]) {
			n++
		}
	}
	return n, nil
}

// procMem returns aggregate process memory in bytes. The optional fourth
// parameter selects the aggregation mode (sum/max/min/avg). RealInUse is
// the closest analogue to RSS on Linux; reported in KB by libperfstat.
func procMem(f procFilter, params []string) (interface{}, error) {
	mode := "sum"
	if len(params) >= 4 && params[3] != "" {
		mode = strings.ToLower(params[3])
	}

	procs, err := perfstat.ProcessStat()
	if err != nil {
		return nil, err
	}
	var (
		matched []int64
		sum     int64
	)
	for i := range procs {
		if !f.matches(&procs[i]) {
			continue
		}
		matched = append(matched, procs[i].RealInUse*1024)
		sum += procs[i].RealInUse * 1024
	}
	if len(matched) == 0 {
		return uint64(0), nil
	}
	switch mode {
	case "sum", "":
		return uint64(sum), nil
	case "max":
		m := matched[0]
		for _, v := range matched[1:] {
			if v > m {
				m = v
			}
		}
		return uint64(m), nil
	case "min":
		m := matched[0]
		for _, v := range matched[1:] {
			if v < m {
				m = v
			}
		}
		return uint64(m), nil
	case "avg":
		return uint64(sum / int64(len(matched))), nil
	}
	return nil, errors.New("Invalid fourth parameter.")
}

type procRecord struct {
	PID     int64  `json:"pid"`
	Name    string `json:"name"`
	UID     int64  `json:"uid"`
	RSS     int64  `json:"rss"`
	VSize   int64  `json:"vsize"`
	Threads int64  `json:"threads"`
}

func procGet(f procFilter) (string, error) {
	procs, err := perfstat.ProcessStat()
	if err != nil {
		return "", err
	}
	out := make([]procRecord, 0, len(procs))
	for i := range procs {
		p := &procs[i]
		if !f.matches(p) {
			continue
		}
		out = append(out, procRecord{
			PID:     p.PID,
			Name:    p.ProcessName,
			UID:     p.UID,
			RSS:     p.RealInUse * 1024,
			VSize:   p.VirtInUse * 1024,
			Threads: p.NumThreads,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
