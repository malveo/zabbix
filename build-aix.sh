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
# Go toolchain in /tmp/dev/go/bin is convention for the LAB4 setup.
export PATH=/opt/freeware/bin:/tmp/dev/go/bin:/usr/bin:/usr/sbin:$PATH
export PKG_CONFIG_PATH=/opt/freeware/lib/pkgconfig

# Force 64-bit C build: AIX gcc defaults to 32-bit XCOFF objects unless
# -maix64 is explicit. This must be in CFLAGS (not just CGO_CFLAGS) so
# that the C-side .a archives are 64-bit and match the Go binary.
export CFLAGS="-maix64 -O2"
export LDFLAGS="-maix64 -L/opt/freeware/lib -Wl,-bbigtoc"

# CGO needs explicit AIX flags. configure.ac propagates these to Makefile.am
# but we set them here too for direct go build invocations during development.
# OpenSSL: IBM's /usr OpenSSL 3.0.16 has OPENSSL_NO_PSK in headers, which
# breaks pkg/tls (PSK is mandatory). We require a custom OpenSSL build at
# /opt/openssl-psk (built once with: ./Configure aix64-gcc no-shared
# --prefix=/opt/openssl-psk && gmake && gmake install_sw).
OPENSSL_PREFIX="${OPENSSL_PREFIX:-/opt/openssl-psk}"
export CGO_CFLAGS="-maix64 -D_THREAD_SAFE -D_LARGE_FILES -I/opt/freeware/include -I${OPENSSL_PREFIX}/include"
export CGO_LDFLAGS="-Wl,-bbigtoc -Wl,-bnoquiet -L${OPENSSL_PREFIX}/lib -L/opt/freeware/lib -L/usr/lib"
# Go 1.16+ refuses -Wl,-bbigtoc as an "invalid" cgo flag because AIX-specific
# linker options are not in the default allowlist. Whitelist explicitly.
export CGO_LDFLAGS_ALLOW='-Wl,-bbigtoc|-Wl,-bnoquiet|-Wl,-bnoentry|-Wl,-bgcbypass|-bbigtoc|-bnoquiet'
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
    --with-openssl="${OPENSSL_PREFIX}" \
    --with-libpcre2=/opt/freeware

echo "[*] Build"
gmake -j2

echo "[+] Done"
ls -l src/go/bin/zabbix_agent2 2>/dev/null && file src/go/bin/zabbix_agent2 || true
