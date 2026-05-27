# Zabbix Agent 2 on IBM AIX 7.2 / 7.3

This branch (`feature/aix-support`) of the Zabbix fork at
`github.com/malveo/zabbix` adds a working build path for Zabbix Agent 2
on `aix/ppc64`. Upstream Zabbix does not support this platform; the
patches in this branch close the gap.

## Status

| Feature                | Status |
|------------------------|--------|
| Build                  | Yes (gcc-13.3, libperfstat, custom OpenSSL with PSK) |
| TLS PSK                | Requires custom OpenSSL build (see below) |
| TLS certificates       | Yes |
| Native AIX plugins     | cpu, memory, netif, vfs/dev, vfs/fs, proc, hw, sw, kernel, uname, uptime |
| C-bridge plugins       | system.localtime, system.boottime, vfs.fs.size, net.tcp.listen, net.udp.listen, system.cpu.load, vm.memory.size (alt path) |
| Disabled               | oracle (godror), sqlite3 persistent buffer (memcache fallback), systemd, smart |

## Prerequisites

Tested on AIX 7.2 TL5 SP11 (LAB4 reference host); 7.3 expected to work
identically — see "Validation" below.

### AIX Toolbox packages

Install via `dnf` (set up the AIX Toolbox repository first if not
present):

```sh
dnf install -y gcc make pkgconf pcre2-devel git rsync \
               autoconf automake libtool m4 perl
```

`gnu-make` ships as `make` once the GNU package is selected by `dnf`;
the AIX `/usr/bin/make` is unsuitable. The build script prepends
`/opt/freeware/bin` to PATH so the GNU tools win.

### Go toolchain

Download the official AIX/ppc64 Go from <https://go.dev/dl/>:

```sh
cd /tmp/dev
curl -sLO https://go.dev/dl/go1.26.3.aix-ppc64.tar.gz
gunzip -c go1.26.3.aix-ppc64.tar.gz | tar xf -
# go binaries live in /tmp/dev/go/bin
```

Any 1.21+ version supports the `aix/ppc64` target; we test against the
latest stable.

### Custom OpenSSL with PSK

IBM's `/usr/lib/libssl.a` defines `OPENSSL_NO_PSK` at the header level,
which breaks `pkg/tls`. Build OpenSSL once into `/opt/openssl-psk`:

```sh
cd /tmp/dev
curl -sLO https://github.com/openssl/openssl/releases/download/openssl-3.5.6/openssl-3.5.6.tar.gz
gunzip -c openssl-3.5.6.tar.gz | tar xf -
cd openssl-3.5.6
perl Configure aix64-gcc shared --prefix=/opt/openssl-psk
gmake -j2
sudo gmake install_sw

# AIX convention: .so members live INSIDE .a archives (see /usr/lib/libssl.a).
# After install_sw we get .a and .so.3 as separate files; merge the .so
# into the .a so dump -H and the runtime loader find libssl.a(libssl.so.3).
cd /opt/openssl-psk/lib
sudo mv libssl64.so.3 libssl.so.3
sudo mv libcrypto64.so.3 libcrypto.so.3
sudo /usr/bin/ar -X64 -q libssl.a libssl.so.3
sudo /usr/bin/ar -X64 -q libcrypto.a libcrypto.so.3
```

`shared` is mandatory: the previous `no-shared` build link-resolves
PSK ciphers from the headers but the runtime falls back to IBM's
/usr/lib OpenSSL which has `OPENSSL_NO_PSK` set — TLS init then
aborts with `no cipher match` even when TLSConnect=unencrypted.

Runtime: `LIBPATH=/opt/openssl-psk/lib` MUST precede `/usr/lib` so
the custom libssl wins over the system one.

## Build

```sh
git clone https://github.com/malveo/zabbix.git
cd zabbix
git checkout feature/aix-support
bash build-aix.sh
```

The script sets `OBJECT_MODE=64`, `AR='/usr/bin/ar -X64'`, the cgo
allowlist for `-Wl,-bbigtoc`, and parks `GOPATH`/`GOCACHE` under `/tmp`
(the default `$HOME/go` typically blows past `/home`'s 1–4 GB quota).

The binary lands at `src/go/bin/zabbix_agent2` (XCOFF 64-bit, ~47 MB).

## Run

```sh
export OBJECT_MODE=64
export LIBPATH=/opt/openssl-psk/lib:/opt/freeware/lib:/opt/oracle/instantclient_19_30:/usr/lib

./src/go/bin/zabbix_agent2 -V
./src/go/bin/zabbix_agent2 -c src/go/conf/zabbix_agent2.conf -t agent.ping
./src/go/bin/zabbix_agent2 -c src/go/conf/zabbix_agent2.conf -t system.cpu.util
```

`LIBPATH` ordering matters: `/opt/freeware` must come before `/usr/lib`
so the `libiconv.so.2` member of `/opt/freeware/lib/libiconv.a` resolves
before the AIX system libiconv (which has no `libiconv_*` prefixed
symbols).

## Validation

Reference smoke test on LAB4 (AIX 7.2 TL5 SP11, 6 GB / 1 disk / 2 NICs):

| Item                             | Result                                 |
|----------------------------------|----------------------------------------|
| `agent.version`                  | `7.4.10`                               |
| `system.uname`                   | `AIX LAB4 2 7 00CC9E184C00`            |
| `system.uptime`                  | seconds since LPAR boot                |
| `system.hw.chassis`              | `IBM IBM,9080-HEU 78C9E18`             |
| `system.sw.os`                   | `AIX 7200-05-11-2546`                  |
| `vm.memory.size[total]`          | `6442450944`                           |
| `vm.memory.size[pused]`          | `89.32`                                |
| `vfs.fs.discovery`               | JSON: `/`, `/usr`, `/var`, `/tmp`, … (jfs2) |
| `vfs.fs.size[/,used]`            | bytes                                  |
| `net.if.discovery`               | JSON: `[{en0}, {lo0}]`                 |
| `net.if.in[en0,bytes]`           | bytes                                  |
| `vfs.dev.discovery`              | JSON: `[{hdisk0,disk}]`                |
| `proc.num`                       | running processes                      |
| `kernel.maxproc`                 | `16384`                                |

`system.cpu.util` returns `ZBX_NODATA` in `-t` test mode (the Collector
runs only in daemon mode). Same caveat as upstream Linux Agent 2.

## Known limitations

* `oracle.*` items disabled — godror requires Oracle Instant Client for
  AIX, which is not bundled. Re-enable per host via local build.
* `proc.cpu.util` returns `UnsupportedMetric`. Implementation pending.
* `system.swap.*` returns `NOTSUPPORTED` — `swap_nix.go` uses
  `syscall.Sysinfo` which is Linux-only. Replacement using
  `perfstat.PagingSpaces()` belongs in a future phase.
* TLS PSK requires the custom OpenSSL build above. With the IBM-shipped
  OpenSSL the agent compiles but PSK paths return errors at runtime.
* Persistent active-check buffer (sqlite3) is not exercised on AIX —
  the agent runs in memcache mode. Buffered metrics are lost on crash.

## Packaging

The `build/aix/rpm/zabbix-agent2.spec` recipe produces an installable
RPM. Build with:

```sh
cd /tmp/dev/zabbix
gtar -czf /tmp/zabbix-agent2-7.4.9.tar.gz --transform 's,^,zabbix-agent2-7.4.9/,' --exclude='.git' .
sudo dnf install -y rpm-build
rpmbuild -ta /tmp/zabbix-agent2-7.4.9.tar.gz
```

The output RPM installs the binary to `/opt/zabbix/sbin`, the config to
`/opt/zabbix/etc`, and registers a `mkitab` entry so the agent starts
at run-level 2.

## Reporting issues

Open a GitHub issue at <https://github.com/malveo/zabbix/issues> with
the failing item key, the agent log (`-d` flag), and the AIX OS level
(`oslevel -s`). Issues that reproduce on the upstream Linux build
should be reported to <https://support.zabbix.com> instead.
