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
#   REMOTE=user@host INSTALL_DIR=/srv/portasplit-monitor ./deploy/deploy.sh
#
set -euo pipefail

REMOTE="${REMOTE:?Set REMOTE=user@host}"
INSTALL_DIR="${INSTALL_DIR:-/opt/portasplit-monitor}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
log() { echo "[deploy] $*"; }

log "ensuring remote dir $INSTALL_DIR exists..."
ssh "$REMOTE" "sudo mkdir -p '$INSTALL_DIR' && sudo chown \"\$(id -un):\$(id -gn)\" '$INSTALL_DIR'"

# Sync the build context. --delete keeps the remote in lockstep with the repo;
# excluded paths (notably .env and the SQLite data) are never touched/removed.
log "syncing source to $REMOTE:$INSTALL_DIR ..."
rsync -az --delete \
  --exclude '.git/' \
  --exclude 'bin/' \
  --exclude 'dist/' \
  --exclude '*.db' --exclude '*.db-wal' --exclude '*.db-shm' \
  --exclude '.env' \
  "$ROOT/" "$REMOTE:$INSTALL_DIR/"

# Ship a local .env ONLY the first time, so server-side secrets are never
# clobbered on later deploys.
if ssh "$REMOTE" "test -f '$INSTALL_DIR/.env'"; then
  log ".env already present on remote, leaving it untouched"
elif [[ -f "$ROOT/.env" ]]; then
  log "uploading local .env (first deploy)..."
  scp -q "$ROOT/.env" "$REMOTE:$INSTALL_DIR/.env"
else
  log "WARNING: no .env on remote and none locally - create $INSTALL_DIR/.env before the app can start"
fi

log "building + starting via docker compose on remote..."
ssh "$REMOTE" "cd '$INSTALL_DIR' && docker compose up -d --build"

log "done. Tail logs with: ssh $REMOTE 'cd $INSTALL_DIR && docker compose logs -f'"
