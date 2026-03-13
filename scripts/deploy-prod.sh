#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.production}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.prod.yml}"
RUN_MIGRATIONS="${RUN_MIGRATIONS:-true}"
export ENV_FILE

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing env file: $ENV_FILE" >&2
  exit 1
fi

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d postgres
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile ops pull api migrate prometheus alertmanager grafana

if [[ "$RUN_MIGRATIONS" == "true" ]]; then
	docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile ops run --rm migrate up
fi

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d api prometheus alertmanager grafana
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps
