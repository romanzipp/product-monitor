#!/usr/bin/env bash
#
# deploy.sh - runs ON YOUR LOCAL MACHINE.
#
# Syncs the repo to a remote host and (re)builds + restarts it with Docker
# Compose. Kubernetes deployment is handled separately via the Helm chart in
# deploy/helm/.
#
# Requirements on the remote host: docker + the docker compose plugin, and an
# SSH user that can run docker (in the `docker` group or via sudo).
#
# Usage:
#   REMOTE=user@host ./deploy/deploy.sh
#   REMOTE=user@host INSTALL_DIR=/srv/product-monitor ./deploy/deploy.sh
#
set -euo pipefail

REMOTE="${REMOTE:?Set REMOTE=user@host}"
INSTALL_DIR="${INSTALL_DIR:-/opt/product-monitor}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
log() { echo "[deploy] $*"; }

log "ensuring remote dir $INSTALL_DIR exists..."
ssh "$REMOTE" "sudo mkdir -p '$INSTALL_DIR' && sudo chown \"\$(id -un):\$(id -gn)\" '$INSTALL_DIR'"

# Sync the build context. --delete keeps the remote in lockstep with the repo;
# excluded paths (.env, config.yaml and the SQLite data) are never touched/removed.
log "syncing source to $REMOTE:$INSTALL_DIR ..."
rsync -az --delete \
  --exclude '.git/' \
  --exclude 'bin/' \
  --exclude 'dist/' \
  --exclude '*.db' --exclude '*.db-wal' --exclude '*.db-shm' \
  --exclude '.env' \
  --exclude 'config.yaml' \
  "$ROOT/" "$REMOTE:$INSTALL_DIR/"

# Ship .env and config.yaml ONLY the first time, so server-side secrets and
# config are never clobbered on later deploys.
upload_once() {
  local name="$1"
  if ssh "$REMOTE" "test -f '$INSTALL_DIR/$name'"; then
    log "$name already present on remote, leaving it untouched"
  elif [[ -f "$ROOT/$name" ]]; then
    log "uploading local $name (first deploy)..."
    scp -q "$ROOT/$name" "$REMOTE:$INSTALL_DIR/$name"
  else
    log "WARNING: no $name on remote and none locally - create $INSTALL_DIR/$name before the app can start"
  fi
}
upload_once .env
upload_once config.yaml

log "building + starting via docker compose on remote..."
ssh "$REMOTE" "cd '$INSTALL_DIR' && docker compose up -d --build"

log "done. Tail logs with: ssh $REMOTE 'cd $INSTALL_DIR && docker compose logs -f'"
