#!/usr/bin/env bash
#
# setup.sh - runs ON THE REMOTE SERVER to install portasplit-monitor as a
# systemd service.
#
# Expects a staging dir (default /tmp/portasplit-monitor-deploy) containing:
#   - portasplit-monitor           (the compiled binary)
#   - portasplit-monitor.service   (the templated systemd unit)
#   - setup.sh                     (this script)
#   - .env                         (optional; installed only if absent)
#
# Usage: sudo bash setup.sh [staging-dir]
#
set -euo pipefail

SRC_DIR="${1:-/tmp/portasplit-monitor-deploy}"
# All overridable via environment for non-default install locations.
INSTALL_DIR="${INSTALL_DIR:-/opt/portasplit-monitor}"
SERVICE_NAME="${SERVICE_NAME:-portasplit-monitor}"
USER_NAME="${USER_NAME:-portasplit-monitor}"
GROUP_NAME="${GROUP_NAME:-portasplit-monitor}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

log() { echo "[setup] $*"; }
die() { echo "[setup][error] $*" >&2; exit 1; }

[[ "$(id -un)" == "root" ]] || die "must run as root (try: sudo bash setup.sh)"
[[ -f "$SRC_DIR/portasplit-monitor" ]] || die "binary not found in $SRC_DIR/portasplit-monitor"
[[ -f "$SRC_DIR/portasplit-monitor.service" ]] || die "unit not found in $SRC_DIR/portasplit-monitor.service"

# 1. Dedicated system user / group.
if ! getent group "$GROUP_NAME" >/dev/null; then
  groupadd --system "$GROUP_NAME"
fi
if ! id -u "$USER_NAME" >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin --gid "$GROUP_NAME" "$USER_NAME"
fi

# 2. Install directory + files.
install -d -o "$USER_NAME" -g "$GROUP_NAME" -m 750 "$INSTALL_DIR"
install -o "$USER_NAME" -g "$GROUP_NAME" -m 755 "$SRC_DIR/portasplit-monitor" "$INSTALL_DIR/portasplit-monitor"

# .env: install from staging only if not already present on the server, so
# manual edits to the server-side .env are never overwritten on redeploy.
if [[ -f "$SRC_DIR/.env" && ! -f "$INSTALL_DIR/.env" ]]; then
  install -o "$USER_NAME" -g "$GROUP_NAME" -m 640 "$SRC_DIR/.env" "$INSTALL_DIR/.env"
  log "installed .env"
fi
[[ -f "$INSTALL_DIR/.env" ]] || die "$INSTALL_DIR/.env missing - create it before starting"
chmod 640 "$INSTALL_DIR/.env"

# 3. systemd unit (substitute the templated placeholders).
sed -e "s#@INSTALL_DIR@#$INSTALL_DIR#g" \
    -e "s#@USER@#$USER_NAME#g" \
    -e "s#@GROUP@#$GROUP_NAME#g" \
    "$SRC_DIR/portasplit-monitor.service" > "$UNIT_FILE"
chmod 644 "$UNIT_FILE"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME" >/dev/null
systemctl restart "$SERVICE_NAME"

# 4. Tidy up any secrets left in the staging dir.
rm -rf "$SRC_DIR"

sleep 1
log "service status:"
systemctl --no-pager --full status "$SERVICE_NAME" || true

cat <<EOF

[setup] Done.
        Binary : $INSTALL_DIR/portasplit-monitor
        Config : $INSTALL_DIR/.env
        Data   : $INSTALL_DIR/portasplit-monitor.db
        Logs   : journalctl -u $SERVICE_NAME -f
        Control: systemctl {start,stop,restart,status} $SERVICE_NAME
EOF
