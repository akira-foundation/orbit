#!/usr/bin/env bash
set -euo pipefail

# Orbit smart orchestration setup for macOS.
#
# After this script:
#   - http://anything.orbit.test  -> Orbit proxy  (no port, no /etc/hosts)
#   - Other *.test domains        -> Herd / Valet untouched
#   - Orbit's own UI               -> stays unprivileged
#
# What it does:
#   1. Adds 127.0.0.2 as a persistent loopback alias on lo0.
#   2. Installs orbit-proxyd: a tiny privileged forwarder that binds
#      127.0.0.2:80 and pipes bytes to 127.0.0.1:2080 (Orbit). Runs as a
#      LaunchDaemon, so Orbit itself never needs root at runtime.
#   3. Configures dnsmasq + /etc/resolver/orbit.test so *.orbit.test resolves
#      to 127.0.0.2.
#
# Idempotent: re-run any time. Requires macOS, Homebrew, sudo.

ALIAS_IP=127.0.0.2
ORBIT_PORT=2080
PUBLIC_PORT=80

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DAEMON_BIN_SRC="${ROOT}/orbit-proxyd"
DAEMON_BIN_DST=/usr/local/libexec/orbit-proxyd

LOOPBACK_PLIST=/Library/LaunchDaemons/com.orbit.loopback.plist
PROXYD_PLIST=/Library/LaunchDaemons/com.orbit.proxyd.plist

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "[orbit] this script is macOS-only" >&2; exit 1
fi
if ! command -v brew &>/dev/null; then
  echo "[orbit] Homebrew is required (https://brew.sh)" >&2; exit 1
fi

# ─── 1. build daemon ─────────────────────────────────────────────────────────
echo "[orbit] 1/5  building orbit-proxyd"
(cd "${ROOT}" && go build -o orbit-proxyd ./cmd/orbit-proxyd)

# ─── 2. loopback alias 127.0.0.2 ─────────────────────────────────────────────
echo "[orbit] 2/5  loopback alias ${ALIAS_IP}"
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

# ─── 3. install proxyd binary + LaunchDaemon ─────────────────────────────────
echo "[orbit] 3/5  installing orbit-proxyd LaunchDaemon"
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

# ─── 4. dnsmasq + resolver ───────────────────────────────────────────────────
echo "[orbit] 4/5  dnsmasq -> *.orbit.test = ${ALIAS_IP}"
if ! brew list dnsmasq &>/dev/null; then brew install dnsmasq; fi
DNSMASQ_CONF="$(brew --prefix)/etc/dnsmasq.conf"
LINE="address=/.orbit.test/${ALIAS_IP}"
if ! grep -qF "${LINE}" "${DNSMASQ_CONF}"; then
  echo "${LINE}" | sudo tee -a "${DNSMASQ_CONF}" >/dev/null
fi
sudo brew services restart dnsmasq

sudo mkdir -p /etc/resolver
echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/orbit.test >/dev/null

sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# ─── 5. sanity checks ────────────────────────────────────────────────────────
echo "[orbit] 5/5  verifying"

ALIAS_OK=$(ifconfig lo0 | grep -c "${ALIAS_IP}" || true)
if [[ "${ALIAS_OK}" -lt 1 ]]; then
  echo "  ! loopback alias missing"; exit 1
fi

DNS_IP=$(dig +short @127.0.0.1 site.orbit.test || true)
if [[ "${DNS_IP}" != "${ALIAS_IP}" ]]; then
  echo "  ! dnsmasq did not return ${ALIAS_IP} (got '${DNS_IP}')"
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
echo "        try:  curl -I http://anything.orbit.test"
echo "              tail -f /var/log/orbit-proxyd.log"
