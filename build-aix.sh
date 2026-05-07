#!/usr/bin/env bash
#
# Build Zabbix Agent 2 on IBM AIX 7.2/7.3 ppc64.
#
# Prerequisites (install via dnf from the AIX Toolbox):
#   - gcc 13+, gnu-make, pkgconf, pcre2-devel
#   - openssl.base from IBM AIX filesets (headers in /usr/include/openssl/)
#   - GNU libiconv from /opt/freeware (NOT the AIX system libiconv in /usr/lib)
#   - Go toolchain for aix/ppc64 (download from https://go.dev/dl/)
#
# This script is intentionally simple: it sets the AIX-specific environment
# (OBJECT_MODE, ar -X64, bigtoc) then runs the standard autotools flow.
#
set -euo pipefail

# 64-bit object mode is required: AIX ar 0707-106 errors stem from mixing
# 32/64-bit objects. Forcing the mode globally avoids the issue.
export OBJECT_MODE=64
export AR="/usr/bin/ar -X64"

# /opt/freeware (AIX Toolbox) ships GNU coreutils with %_d date, GNU make,
# pkg-config, and libiconv with libiconv_* prefixed symbols. PATH ordering
# matters: AIX system /usr/bin must come AFTER for libiconv to resolve.
export PATH=/opt/freeware/bin:/usr/bin:/usr/sbin:$PATH
export PKG_CONFIG_PATH=/opt/freeware/lib/pkgconfig

# CGO needs explicit AIX flags. configure.ac propagates these to Makefile.am
# but we set them here too for direct go build invocations during development.
export CGO_CFLAGS="-maix64 -D_THREAD_SAFE -D_LARGE_FILES -I/opt/freeware/include"
export CGO_LDFLAGS="-Wl,-bbigtoc -Wl,-bnoquiet -L/opt/freeware/lib -L/usr/lib"
export GOOS=aix
export GOARCH=ppc64

if [ ! -f configure ]; then
    echo "[*] Bootstrap (autoreconf)"
    ./bootstrap.sh
fi

echo "[*] Configure"
./configure \
    --enable-agent2 \
    --prefix=/opt/zabbix \
    --with-openssl=/usr \
    --with-libpcre2=/opt/freeware

echo "[*] Build"
gmake -j2

echo "[+] Done"
ls -l src/go/bin/zabbix_agent2 2>/dev/null && file src/go/bin/zabbix_agent2 || true
