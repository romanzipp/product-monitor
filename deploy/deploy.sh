#!/usr/bin/env bash
#
# deploy.sh - runs ON YOUR LOCAL MACHINE.
#
# Cross-compiles a static linux binary, uploads it (plus the unit + a local
# .env if present) to a remote host, and invokes setup.sh there via SSH to
# install it as a systemd service. Kubernetes deployment is handled separately
# via the Helm chart in deploy/helm/.
#
# Usage:
#   REMOTE=user@host ./deploy/deploy.sh
#   REMOTE=user@host GOARCH=arm64 ./deploy/deploy.sh
#
set -euo pipefail

REMOTE="${REMOTE:?Set REMOTE=user@host}"
ARCH="${GOARCH:-amd64}"
REMOTE_TMP="${REMOTE_TMP:-/tmp/portasplit-monitor-deploy}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
log() { echo "[deploy] $*"; }

log "building linux/$ARCH (CGO disabled, static)..."
mkdir -p "$ROOT/bin"
(
  cd "$ROOT"
  CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
    go build -trimpath -ldflags "-s -w" -o "bin/portasplit-monitor-linux-$ARCH" ./cmd/portasplit-monitor
)

log "preparing remote staging dir $REMOTE_TMP..."
ssh "$REMOTE" "rm -rf '$REMOTE_TMP' && mkdir -p '$REMOTE_TMP'"

log "uploading binary + service unit + setup script..."
scp -q "$ROOT/bin/portasplit-monitor-linux-$ARCH" "$REMOTE:$REMOTE_TMP/portasplit-monitor"
scp -q "$ROOT/deploy/portasplit-monitor.service" "$REMOTE:$REMOTE_TMP/portasplit-monitor.service"
scp -q "$ROOT/deploy/setup.sh" "$REMOTE:$REMOTE_TMP/setup.sh"

# Ship a local .env ONLY the first time, so server-side secrets are never
# clobbered on later deploys. setup.sh also guards against overwriting.
if [[ -f "$ROOT/.env" ]]; then
  log "uploading local .env (installed only if absent on server)..."
  scp -q "$ROOT/.env" "$REMOTE:$REMOTE_TMP/.env"
fi

log "running setup.sh on remote..."
ssh "$REMOTE" "sudo bash '$REMOTE_TMP/setup.sh' '$REMOTE_TMP'"

log "done."
