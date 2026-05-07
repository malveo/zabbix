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

package cpu

import (
	"github.com/power-devops/perfstat"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

// Plugin -
type Plugin struct {
	plugin.Base
	cpus []*cpuUnit
}

func init() {
	err := plugin.RegisterMetrics(
		&impl, pluginName,
		"system.cpu.discovery", "List of detected CPUs/CPU cores, used for low-level discovery.",
		"system.cpu.num", "Number of CPUs.",
		"system.cpu.util", "CPU utilization percentage.",
	)
	if err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

// Period — sampling interval in seconds; matches Linux backend.
func (*Plugin) Period() int { return 1 }

// getCPULoad is provided here as a stub; system.cpu.load comes from the
// C agent path (zbxlib bridge), not from this Go plugin.
func (*Plugin) getCPULoad(_ []string) (any, error) {
	return nil, plugin.UnsupportedMetricError
}

// Collect snapshots all CPU counters via libperfstat into the per-CPU
// history ring used by counterAverage(). The aggregate "all CPUs" entry
// lives at index 0; per-CPU entries follow.
func (p *Plugin) Collect() error {
	for _, cpu := range p.cpus {
		cpu.status = cpuStatusOffline
	}

	// Aggregate counters first.
	total, err := perfstat.CpuTotalStat()
	if err != nil {
		return err
	}
	if len(p.cpus) > 0 {
		all := p.cpus[0]
		all.status = cpuStatusOnline
		slot := &all.history[all.tail]
		slot.counters[counterUser] = uint64(total.User)
		slot.counters[counterSystem] = uint64(total.Sys)
		slot.counters[counterIdle] = uint64(total.Idle)
		slot.counters[counterIowait] = uint64(total.Wait)
		// nice/irq/softirq/steal/guest unavailable on AIX — leave at zero.
		if all.tail = all.tail.inc(); all.tail == all.head {
			all.head = all.head.inc()
		}
	}

	// Per-CPU counters.
	cpus, err := perfstat.CpuStat()
	if err != nil {
		return err
	}
	for i, c := range cpus {
		p.addCpu(i)
		// p.cpus[i+1] holds CPU index i.
		if i+1 >= len(p.cpus) {
			continue
		}
		unit := p.cpus[i+1]
		unit.status = cpuStatusOnline
		slot := &unit.history[unit.tail]
		slot.counters[counterUser] = uint64(c.User)
		slot.counters[counterSystem] = uint64(c.Sys)
		slot.counters[counterIdle] = uint64(c.Idle)
		slot.counters[counterIowait] = uint64(c.Wait)
		if unit.tail = unit.tail.inc(); unit.tail == unit.head {
			unit.head = unit.head.inc()
		}
	}
	return nil
}

func (p *Plugin) addCpu(index int) {
	if p == nil || p.cpus == nil {
		return
	}
	// p.cpus is sized as numCPUConf+1; only grow if perfstat reports
	// more CPUs than detected at startup (CPU hot-add scenarios).
	for index+1 >= len(p.cpus) {
		p.cpus = append(p.cpus,
			&cpuUnit{index: len(p.cpus) - 1, status: cpuStatusOffline})
	}
}

func (*Plugin) getCounterAverage(cpu *cpuUnit, counter cpuCounter, period historyIndex) any {
	return cpu.counterAverage(counter, period, 1)
}

func numCPUConf() int {
	cpus, err := perfstat.CpuStat()
	if err != nil {
		return 0
	}
	return len(cpus)
}

func numCPUOnline() int {
	// libperfstat returns only the CPUs that are configured AND online.
	return numCPUConf()
}

func (p *Plugin) Start() {
	p.cpus = p.newCpus(numCPUConf())
}

func (p *Plugin) Stop() {
	p.cpus = nil
}

func (p *Plugin) Export(key string, params []string, ctx plugin.ContextProvider) (result interface{}, err error) {
	if p.cpus == nil || p.cpus[0].head == p.cpus[0].tail {
		return
	}
	switch key {
	case "system.cpu.discovery":
		return p.getCpuDiscovery(params)
	case "system.cpu.num":
		return p.getCpuNum(params)
	case "system.cpu.util":
		return p.getCpuUtil(params)
	default:
		return nil, plugin.UnsupportedMetricError
	}
}
