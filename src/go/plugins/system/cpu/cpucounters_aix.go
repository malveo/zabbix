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

// AIX exposes 4 cycle counters via libperfstat: user, sys, idle, wait.
// The remaining Linux counters (nice, irq, softirq, steal, guest) are
// not available — counterByType still recognises the names so that
// existing item configurations don't error out, but the underlying
// counter slot stays at zero and utilization for those types reports 0%.

const (
	counterUnknown cpuCounter = iota - 1
	counterUser
	counterSystem
	counterIdle
	counterIowait
	counterNice
	counterIrq
	counterSoftirq
	counterSteal
	counterGcpu
	counterGnice
	counterNum
)

type cpuCounters struct {
	counters [counterNum]uint64
}

func counterByType(name string) cpuCounter {
	switch name {
	case "", "user":
		return counterUser
	case "system":
		return counterSystem
	case "idle":
		return counterIdle
	case "iowait":
		return counterIowait
	case "nice":
		return counterNice
	case "interrupt":
		return counterIrq
	case "softirq":
		return counterSoftirq
	case "steal":
		return counterSteal
	case "guest":
		return counterGcpu
	case "guest_nice":
		return counterGnice
	default:
		return counterUnknown
	}
}

func (c *cpuUnit) counterAverage(counter cpuCounter, period historyIndex, _ int) (result interface{}) {
	if c.head == c.tail {
		return
	}

	totalnum := c.tail - c.head
	if totalnum < 0 {
		totalnum += maxHistory
	}
	if totalnum < 2 {
		return
	}
	if totalnum-1 < period {
		period = totalnum - 1
	}

	tail := &c.history[c.tail.dec()]
	head := &c.history[c.tail.sub(period+1)]

	var value, total uint64
	for i := 0; i < len(tail.counters); i++ {
		if tail.counters[i] > head.counters[i] {
			total += tail.counters[i] - head.counters[i]
		}
	}
	if total == 0 {
		return
	}

	if tail.counters[counter] > head.counters[counter] {
		value = tail.counters[counter] - head.counters[counter]
	}
	return float64(value) * 100 / float64(total)
}
