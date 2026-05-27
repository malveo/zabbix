#!/usr/bin/env bash
#
# Build Zabbix Agent 2 on IBM AIX 7.2/7.3 ppc64.
#
# Most of the AIX build configuration lives in configure.ac (aix* branch)
# and src/go/Makefile.am (if AIX block). This script only carries
# context that cannot be expressed in autoconf:
#
#   * shell PATH ordering — Make cannot reorder the user's PATH before
#     ./configure runs, and GNU make/coreutils from /opt/freeware/bin
#     must win over the AIX defaults so $(shell ...) in Makefile.am
#     behaves like Linux.
#   * GOPATH / GOCACHE redirection — /home is typically 1–4 GB on AIX
#     and exhausts mid-build; park them under /tmp instead.
#   * pkg-config search path.
#
# Prerequisites (install once per host via dnf from the AIX Toolbox):
#   gcc 13+, make, pkg-config, pcre2-devel, autoconf, automake,
#   libtool, m4, perl, curl, rsync, git, unzip, libzstd.
# Plus:
#   * Go toolchain for aix/ppc64 at /tmp/dev/go (download from go.dev/dl/)
#   * Custom OpenSSL 3.5.6 with PSK at /opt/openssl-psk (see INSTALL_AIX.md)
#   * Oracle Instant Client 19.30 at /opt/oracle/instantclient_19_30
#
set -euo pipefail

# GNU tools first, Go toolchain reachable.
export PATH=/opt/freeware/bin:/tmp/dev/go/bin:/usr/bin:/usr/sbin:$PATH
export PKG_CONFIG_PATH=/opt/freeware/lib/pkgconfig

# Park Go state on /tmp (LAB hosts have small /home filesystems).
export GOPATH="${GOPATH:-/tmp/dev/gopath}"
export GOCACHE="${GOCACHE:-/tmp/dev/gocache}"
export GOMODCACHE="${GOMODCACHE:-${GOPATH}/pkg/mod}"
mkdir -p "$GOPATH" "$GOCACHE" "$GOMODCACHE"

# Generate configure if absent.
if [ ! -f configure ]; then
    echo "[*] Bootstrap (autoreconf)"
    ./bootstrap.sh
fi

# Configure. CFLAGS, LDFLAGS, CGO_*, AR, OBJECT_MODE, GOOS, GOARCH,
# CGO_LDFLAGS_ALLOW are all set inside configure.ac (aix* branch) and
# propagated via @AGENT2_*@ Make substitutions.
echo "[*] Configure"
./configure \
    --enable-agent2 \
    --prefix=/opt/zabbix \
    --with-openssl="${OPENSSL_PREFIX:-/opt/openssl-psk}" \
    --with-libpcre2=/opt/freeware \
    --with-oracle-icr="${ICR_PREFIX:-/opt/oracle/instantclient_19_30}"

echo "[*] Build"
gmake -j2

echo "[+] Done"
ls -l src/go/bin/zabbix_agent2 2>/dev/null && file src/go/bin/zabbix_agent2 || true
