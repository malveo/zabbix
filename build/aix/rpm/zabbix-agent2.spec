# Zabbix Agent 2 — AIX 7.2/7.3 ppc64
#
# Builds and packages the agent2 binary together with the bundled
# config and an /etc/rc.d/init.d entry. Targets AIX rpm (rpm-4.x from
# the IBM AIX Toolbox / dnf).
#
# Prerequisite at build time:
#   - /opt/openssl-psk built per INSTALL_AIX.md
#   - Go toolchain on PATH
#   - PATH=/opt/freeware/bin:/usr/bin first

%define _prefix       /opt/zabbix
%define _sysconfdir   %{_prefix}/etc
%define _sbindir      %{_prefix}/sbin
%define _datadir      %{_prefix}/share

%global _build_id_links none
%global debug_package %{nil}

Name:           zabbix-agent2
Version:        7.4.9
Release:        aix.14%{?dist}
Summary:        Zabbix Agent 2 for IBM AIX 7.x
License:        AGPLv3
URL:            https://github.com/malveo/zabbix
Source0:        %{name}-%{version}.tar.gz
ExclusiveArch:  ppc

BuildRequires:  gcc >= 13
BuildRequires:  pkgconf
BuildRequires:  pcre2-devel
BuildRequires:  make

%description
Zabbix monitoring agent (Go-based, hybrid cgo) ported to IBM AIX
7.2 and 7.3 on ppc64. Provides ~30 native item keys via libperfstat
plus the C-bridge metrics inherited from the classic agent.

This is a downstream community build — for upstream agent (Linux,
Windows, macOS) use https://www.zabbix.com/download_agents.

%prep
%setup -q

%build
export OBJECT_MODE=64
export PATH=/opt/freeware/bin:/usr/bin:/usr/sbin:$PATH
bash build-aix.sh

%install
rm -rf %{buildroot}
install -d %{buildroot}%{_sbindir}
install -d %{buildroot}%{_sysconfdir}/zabbix_agent2.d/plugins.d
install -d %{buildroot}/etc/rc.d/init.d
install -d %{buildroot}%{_datadir}/doc

# Binary — note that build-aix.sh emits to src/go/bin (a file, not a
# directory) on first run; both names are handled.
if [ -d src/go/bin ]; then
    install -m 0755 src/go/bin/zabbix_agent2 %{buildroot}%{_sbindir}/zabbix_agent2
else
    install -m 0755 src/go/bin %{buildroot}%{_sbindir}/zabbix_agent2
fi

# Config and plugin templates
install -m 0644 src/go/conf/zabbix_agent2.conf  %{buildroot}%{_sysconfdir}/zabbix_agent2.conf
cp -r src/go/conf/zabbix_agent2.d/plugins.d/*   %{buildroot}%{_sysconfdir}/zabbix_agent2.d/plugins.d/ || true

# rc init script — minimal start/stop wrapper using AIX startsrc/stopsrc
cat > %{buildroot}/etc/rc.d/init.d/zabbix_agent2 <<'INIT'
#!/bin/sh
# zabbix_agent2 init script for AIX
PIDFILE=/var/run/zabbix/zabbix_agent2.pid
BIN=%{_sbindir}/zabbix_agent2
CONF=%{_sysconfdir}/zabbix_agent2.conf
export LIBPATH=/opt/openssl-psk/lib:/opt/freeware/lib:/opt/oracle/instantclient_19_30:/usr/lib
case "$1" in
    start)
        mkdir -p /var/run/zabbix
        $BIN -c $CONF >/dev/null 2>&1 &
        echo $! > $PIDFILE
        ;;
    stop)
        if [ -s $PIDFILE ]; then
            kill `cat $PIDFILE` 2>/dev/null
            rm -f $PIDFILE
        fi
        ;;
    restart)
        $0 stop
        sleep 1
        $0 start
        ;;
    status)
        if [ -s $PIDFILE ] && kill -0 `cat $PIDFILE` 2>/dev/null; then
            echo "zabbix_agent2 running (pid `cat $PIDFILE`)"
        else
            echo "zabbix_agent2 stopped"; exit 1
        fi
        ;;
    *) echo "Usage: $0 {start|stop|restart|status}"; exit 1 ;;
esac
INIT
chmod 0755 %{buildroot}/etc/rc.d/init.d/zabbix_agent2

# Documentation
install -m 0644 INSTALL_AIX.md %{buildroot}%{_datadir}/doc/INSTALL_AIX.md
install -m 0644 ChangeLog       %{buildroot}%{_datadir}/doc/ChangeLog || true

%files
%defattr(-,root,system,-)
%{_sbindir}/zabbix_agent2
%config(noreplace) %{_sysconfdir}/zabbix_agent2.conf
%{_sysconfdir}/zabbix_agent2.d/plugins.d/
/etc/rc.d/init.d/zabbix_agent2
%doc %{_datadir}/doc/INSTALL_AIX.md

%post
# Register with SRC inittab so the agent restarts on boot. -h o means
# only run once at run-level 2 (matches Zabbix Linux init pattern).
mkitab "zabbix_agent2:2:once:/etc/rc.d/init.d/zabbix_agent2 start" 2>/dev/null || true

%preun
rmitab zabbix_agent2 2>/dev/null || true
/etc/rc.d/init.d/zabbix_agent2 stop 2>/dev/null || true

%changelog
* Fri May 08 2026 Mauro Malvestio <github@malvestio.net> - 7.4.9-aix.4
- Phase 5: proc plugin (proc.num/mem/get), systemd disabled, RPM spec, INSTALL_AIX.md.
* Thu May 07 2026 Mauro Malvestio <github@malvestio.net> - 7.4.9-aix.3
- Phase 4: perfstat plugins (cpu, memory, netif, vfs/dev, hw, sw).
* Thu May 07 2026 Mauro Malvestio <github@malvestio.net> - 7.4.9-aix.2
- Phase 3: real uptime, kernel, vfs.fs.discovery via mount.
* Thu May 07 2026 Mauro Malvestio <github@malvestio.net> - 7.4.9-aix.1
- Phase 2: toolchain + autotools, agent2 builds and runs on AIX 7.2.
* Thu May 07 2026 Mauro Malvestio <github@malvestio.net> - 7.4.9-aix.0
- Phase 1: skeleton glue layer.
