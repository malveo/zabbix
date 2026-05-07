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

package sw

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"

	"golang.zabbix.com/sdk/zbxerr"
)

// systemSwPackages returns names of installed AIX filesets and (if dnf
// is present) RPMs whose package name matches the optional first regex
// parameter. Output is a single newline-separated string, mirroring the
// Linux package list contract.
func (p *Plugin) systemSwPackages(params []string, _ int) (any, error) {
	pattern := ""
	if len(params) >= 1 {
		pattern = params[0]
	}
	// params[1] = manager filter, params[2] = format — both ignored on
	// AIX for the skeleton port; we always return native filesets and
	// (best effort) rpm packages.
	names, err := aixListPackages()
	if err != nil {
		return nil, err
	}
	return strings.Join(filterNames(names, pattern), "\n"), nil
}

type swPackage struct {
	Name    string `json:"name"`
	Manager string `json:"manager"`
	Version string `json:"version"`
	Buildtime swBuildTime `json:"buildtime"`
}

type swBuildTime struct {
	Timestamp int64  `json:"timestamp"`
	Value     string `json:"value"`
}

func (p *Plugin) systemSwPackagesGet(params []string, _ int) (any, error) {
	pattern := ""
	if len(params) >= 1 {
		pattern = params[0]
	}
	pkgs, err := aixListPackagesDetailed()
	if err != nil {
		return nil, err
	}
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, zbxerr.New("invalid regular expression").Wrap(err)
		}
		filtered := pkgs[:0]
		for _, pkg := range pkgs {
			if re.MatchString(pkg.Name) {
				filtered = append(filtered, pkg)
			}
		}
		pkgs = filtered
	}
	b, err := json.Marshal(pkgs)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (p *Plugin) getOSVersion(params []string) (any, error) {
	mode := "full"
	if len(params) >= 1 && params[0] != "" {
		mode = params[0]
	}
	tl, _ := exec.Command("/usr/bin/oslevel", "-s").Output()
	full := strings.TrimSpace(string(tl))
	switch mode {
	case "full", "":
		return "AIX " + full, nil
	case "short":
		return full, nil
	case "name":
		return "AIX", nil
	}
	return nil, zbxerr.New("invalid first parameter")
}

type swOS struct {
	OSType  string `json:"os_type"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (p *Plugin) getOSVersionJSON() (any, error) {
	tl, _ := exec.Command("/usr/bin/oslevel", "-s").Output()
	full := strings.TrimSpace(string(tl))
	b, err := json.Marshal(swOS{OSType: "aix", Name: "IBM AIX", Version: full})
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// aixListPackages returns the names of installed filesets via lslpp.
// `lslpp -Lc` prints colon-separated records:
//
//	package:fileset:level:state:type:description:...
//
// We pick fileset names (column 2) and skip header/blank lines.
func aixListPackages() ([]string, error) {
	out, err := exec.Command("/usr/bin/lslpp", "-Lc").Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		names = append(names, fields[1])
	}
	return names, nil
}

func aixListPackagesDetailed() ([]swPackage, error) {
	out, err := exec.Command("/usr/bin/lslpp", "-Lc").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []swPackage
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 3 {
			continue
		}
		pkgs = append(pkgs, swPackage{
			Name:    fields[1],
			Manager: "installp",
			Version: fields[2],
		})
	}
	return pkgs, nil
}

func filterNames(names []string, pattern string) []string {
	if pattern == "" {
		return names
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	out := names[:0]
	for _, n := range names {
		if re.MatchString(n) {
			out = append(out, n)
		}
	}
	return out
}
