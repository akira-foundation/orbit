#!/usr/bin/env bash
set -euo pipefail

# Orbit smart-orchestration setup for macOS.
#
# After this script:
#   http://anything.orbit.test  -> Orbit proxy on 127.0.0.2:80
#   *.test (Herd / Valet)        -> untouched on 127.0.0.1
#
# Steps (no Homebrew, no dnsmasq dependency):
#   1. Add 127.0.0.2 as a persistent loopback alias on lo0.
#   2. Install orbit-proxyd: privileged daemon that
#        - forwards 127.0.0.2:80 -> 127.0.0.1:2080 (the Orbit app)
#        - serves DNS on 127.0.0.2:53 answering *.orbit.test = 127.0.0.2
#   3. Add /etc/resolver/orbit.test -> nameserver 127.0.0.2 so macOS only
#      routes .orbit.test queries to us (Herd's resolver for .test stays).
#
# Idempotent: re-run any time. Requires macOS, sudo.

ALIAS_IP=127.0.0.2
ORBIT_PORT=2080
ORBIT_TLS_PORT=2443
PUBLIC_PORT=80
PUBLIC_TLS_PORT=443
DNS_PORT=53
SUFFIX=orbit.test

DAEMON_BIN_SRC="${ORBIT_PROXYD_BIN:-}"
if [[ -z "${DAEMON_BIN_SRC}" || ! -x "${DAEMON_BIN_SRC}" ]]; then
  echo "[orbit] orbit-proxyd binary not provided (set ORBIT_PROXYD_BIN)" >&2
  exit 1
fi
DAEMON_BIN_DST=/usr/local/libexec/orbit-proxyd

LOOPBACK_PLIST=/Library/LaunchDaemons/com.orbit.loopback.plist
PROXYD_PLIST=/Library/LaunchDaemons/com.orbit.proxyd.plist

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[orbit] this script is macOS-only" >&2; exit 1
fi

# ─── 1. loopback alias 127.0.0.2 ─────────────────────────────────────────────
echo "[orbit] 1/3  loopback alias ${ALIAS_IP}"
sudo ifconfig lo0 alias ${ALIAS_IP} up || true

sudo tee "${LOOPBACK_PLIST}" >/dev/null <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.orbit.loopback</string>
  <key>RunAtLoad</key><true/>
  <key>ProgramArguments</key>
  <array>
    <string>/sbin/ifconfig</string><string>lo0</string><string>alias</string>
    <string>${ALIAS_IP}</string><string>up</string>
  </array>
</dict>
</plist>
EOF
sudo chown root:wheel "${LOOPBACK_PLIST}"
sudo chmod 644 "${LOOPBACK_PLIST}"
sudo launchctl bootstrap system "${LOOPBACK_PLIST}" 2>/dev/null || \
  sudo launchctl load -w "${LOOPBACK_PLIST}" 2>/dev/null || true

# ─── 2. orbit-proxyd (TCP + DNS) ─────────────────────────────────────────────
echo "[orbit] 2/3  installing orbit-proxyd LaunchDaemon"
sudo install -d -m 755 /usr/local/libexec
sudo install -m 755 "${DAEMON_BIN_SRC}" "${DAEMON_BIN_DST}"

sudo tee "${PROXYD_PLIST}" >/dev/null <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.orbit.proxyd</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ProgramArguments</key>
  <array>
    <string>${DAEMON_BIN_DST}</string>
    <string>-src</string><string>${ALIAS_IP}:${PUBLIC_PORT}</string>
    <string>-dst</string><string>127.0.0.1:${ORBIT_PORT}</string>
    <string>-tls-src</string><string>${ALIAS_IP}:${PUBLIC_TLS_PORT}</string>
    <string>-tls-dst</string><string>127.0.0.1:${ORBIT_TLS_PORT}</string>
    <string>-dns</string><string>${ALIAS_IP}:${DNS_PORT}</string>
    <string>-suffix</string><string>${SUFFIX}</string>
    <string>-answer</string><string>${ALIAS_IP}</string>
  </array>
  <key>StandardOutPath</key><string>/var/log/orbit-proxyd.log</string>
  <key>StandardErrorPath</key><string>/var/log/orbit-proxyd.err.log</string>
</dict>
</plist>
EOF
sudo chown root:wheel "${PROXYD_PLIST}"
sudo chmod 644 "${PROXYD_PLIST}"
sudo launchctl bootout system "${PROXYD_PLIST}" 2>/dev/null || true
sudo launchctl bootstrap system "${PROXYD_PLIST}"

# ─── 3. resolver -> 127.0.0.2 ────────────────────────────────────────────────
echo "[orbit] 3/3  /etc/resolver/${SUFFIX} -> ${ALIAS_IP}"
sudo mkdir -p /etc/resolver
echo "nameserver ${ALIAS_IP}" | sudo tee /etc/resolver/${SUFFIX} >/dev/null

sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# ─── verify ──────────────────────────────────────────────────────────────────
sleep 1

ALIAS_OK=$(ifconfig lo0 | grep -c "${ALIAS_IP}" || true)
if [[ "${ALIAS_OK}" -lt 1 ]]; then
  echo "  ! loopback alias missing"; exit 1
fi

DNS_IP=$(dig +short +time=1 +tries=1 @${ALIAS_IP} site.${SUFFIX} || true)
if [[ "${DNS_IP}" != "${ALIAS_IP}" ]]; then
  echo "  ! orbit DNS did not return ${ALIAS_IP} (got '${DNS_IP}')"
fi

if ! sudo lsof -nP -iTCP:${PUBLIC_PORT} -sTCP:LISTEN 2>/dev/null | grep -q "${ALIAS_IP}"; then
  echo
  echo "  WARNING: nothing is listening on ${ALIAS_IP}:${PUBLIC_PORT}."
  echo "  Most likely cause: Laravel Herd is binding 0.0.0.0:${PUBLIC_PORT}, which"
  echo "  shadows ${ALIAS_IP}:${PUBLIC_PORT}. Open Herd Settings and enable"
  echo "  'Bind to localhost only' (or quit Herd), then re-run this script."
fi

echo
echo "[orbit] done."
echo "        try:  curl -I http://anything.${SUFFIX}"
echo "              tail -f /var/log/orbit-proxyd.log"
