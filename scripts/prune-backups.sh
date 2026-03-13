#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.production}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE" >&2
  exit 1
fi

set -a
source "$ENV_FILE"
set +a

BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"

if [[ ! -d "$BACKUP_DIR" ]]; then
  echo "Backup dir does not exist: $BACKUP_DIR"
  exit 0
fi

find "$BACKUP_DIR" -type f -name 'marketplace_*.dump' -mtime +"$RETENTION_DAYS" -print -delete
