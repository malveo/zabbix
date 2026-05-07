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

package vfsfs

import (
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

func init() {
	err := plugin.RegisterMetrics(
		&impl, "VfsFs",
		"vfs.fs.discovery", "List of mounted filesystems. Used for low-level discovery.",
		"vfs.fs.get", "List of mounted filesystems with statistics.",
		"vfs.fs.size", "Disk space in bytes or in percentage from total.",
		"vfs.fs.inode", "Disk space in bytes or in percentage from total.",
	)
	if err != nil {
		panic(errs.Wrap(err, "failed to register metrics"))
	}
}

func (p *Plugin) getFsInfoStats() (data []*FsInfoNew, err error) {
	allData, err := p.getFsInfo()
	if err != nil {
		return nil, err
	}

	fsmap := make(map[string]*FsInfoNew)
	fsStatCaller := p.newFSCaller(getFsStats, len(allData))
	fsInodeCaller := p.newFSCaller(getFsInode, len(allData))

	for _, info := range allData {
		bytes, err := fsStatCaller.run(*info.FsName)
		if err != nil {
			p.Debugf(`cannot discern stats for the mount %s: %s`, *info.FsName, err.Error())
			continue
		}

		inodes, err := fsInodeCaller.run(*info.FsName)
		if err != nil {
			p.Debugf(`cannot discern inode for the mount %s: %s`, *info.FsName, err.Error())
			continue
		}

		fsmap[*info.FsName+*info.FsType] = &FsInfoNew{info.FsName, info.FsType, nil, nil, bytes, inodes, info.FsOptions}
	}

	allData, err = p.getFsInfo()
	if err != nil {
		return nil, err
	}

	for _, info := range allData {
		if fsInfo, ok := fsmap[*info.FsName+*info.FsType]; ok {
			data = append(data, fsInfo)
		}
	}

	return
}

// getFsInfo reads /etc/filesystems is rare on AIX; the canonical way to
// enumerate active mounts is the `mount` command. Output format:
//
//	  node       mounted        mounted over    vfs       date        options
//	  -------- ---------------  ---------------  ------ ------------ -------
//	           /dev/hd4         /                jfs2   May 07 15:57 rw,log=/dev/hd8
//	           /dev/hd2         /usr             jfs2   May 07 15:57 rw,log=/dev/hd8
//
// Columns are space-separated; the first node column may be empty for
// local mounts. We split on whitespace and pick by position; the
// mount-point and vfs type are stable across AIX releases.
func (p *Plugin) getFsInfo() (data []*FsInfo, err error) {
	out, err := exec.Command("/usr/sbin/mount").Output()
	if err != nil {
		return nil, err
	}

	for i, line := range strings.Split(string(out), "\n") {
		// Skip header rows and blank lines.
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		// Layout: node? device mountpoint vfs date... options
		// When node is empty (local mount) the first field is the device.
		var dev, mnt, vfs, opts string
		switch {
		case len(fields) >= 5 && strings.HasPrefix(fields[0], "/"):
			// Local mount: device, mountpoint, vfs, date(3 cols), options
			dev = fields[0]
			mnt = fields[1]
			vfs = fields[2]
			if len(fields) >= 7 {
				opts = strings.Join(fields[6:], " ")
			}
		case len(fields) >= 6:
			// Remote mount: node, device, mountpoint, vfs, date(3), options
			dev = fields[1]
			mnt = fields[2]
			vfs = fields[3]
			if len(fields) >= 8 {
				opts = strings.Join(fields[7:], " ")
			}
		default:
			continue
		}

		_ = dev
		// Copy strings: FsInfo holds pointers; closure-trap on loop var would
		// otherwise alias every entry to the same backing storage.
		mntCopy := mnt
		vfsCopy := vfs
		optsCopy := opts
		data = append(data, &FsInfo{FsName: &mntCopy, FsType: &vfsCopy, FsOptions: &optsCopy})
	}

	return data, nil
}

func getFsStats(path string) (stats *FsStats, err error) {
	var pused float64

	fs := unix.Statfs_t{}
	err = unix.Statfs(path, &fs)
	if err != nil {
		return nil, err
	}

	var available uint64
	if fs.Bavail > 0 {
		available = uint64(fs.Bavail)
	}

	total := uint64(fs.Blocks) * uint64(fs.Bsize)
	free := available * uint64(fs.Bsize)
	used := (uint64(fs.Blocks) - uint64(fs.Bfree)) * uint64(fs.Bsize)
	pfree := float64(uint64(fs.Blocks) - uint64(fs.Bfree) + uint64(fs.Bavail))

	if pfree > 0 {
		pfree = 100.00 * float64(available) / pfree
		pused = 100 - pfree
	} else {
		pfree = 0
		pused = 0
	}

	stats = &FsStats{
		Total: total,
		Free:  free,
		Used:  used,
		PFree: pfree,
		PUsed: pused,
	}

	return
}

func getFsInode(path string) (stats *FsStats, err error) {
	var pfree, pused float64

	fs := unix.Statfs_t{}
	err = unix.Statfs(path, &fs)
	if err != nil {
		return nil, err
	}

	total := uint64(fs.Files)
	free := uint64(fs.Ffree)
	used := total - free

	if 0 < total {
		pfree = 100 * float64(free) / float64(total)
		pused = 100 * float64(total-free) / float64(total)
	} else {
		pfree = 100.0
		pused = 0.0
	}

	stats = &FsStats{
		Total: total,
		Free:  free,
		Used:  used,
		PFree: pfree,
		PUsed: pused,
	}

	return
}
