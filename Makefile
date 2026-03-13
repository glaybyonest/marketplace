SHELL := /bin/bash

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.26.0
DB_URL ?= $(DATABASE_URL)
TEST_DB_URL ?= $(TEST_DATABASE_URL)

.PHONY: up down build run migrate-up migrate-down seed test lint fmt ci frontend-install frontend-dev openapi-check
.PHONY: prod-config prod-up prod-down prod-migrate prod-deploy backup restore prune-backups

up:
	docker compose up -d --build

down:
	docker compose down -v

build:
	go build ./cmd/api

run:
	go run ./cmd/api

migrate-up:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" up

migrate-down:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" down

seed:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" up

test:
	go test ./... -race -coverprofile=coverage.out

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

ci: fmt lint test

openapi-check:
	go test ./apidocs

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

prod-config:
	ENV_FILE=.env.production.example docker compose --env-file .env.production.example -f docker-compose.prod.yml config >/dev/null

prod-up:
	ENV_FILE=.env.production bash scripts/deploy-prod.sh

prod-down:
	ENV_FILE=.env.production docker compose --env-file .env.production -f docker-compose.prod.yml down

prod-migrate:
	ENV_FILE=.env.production docker compose --env-file .env.production -f docker-compose.prod.yml --profile ops run --rm migrate up

prod-deploy:
	ENV_FILE=.env.production RUN_MIGRATIONS=true bash scripts/deploy-prod.sh

backup:
	ENV_FILE=.env.production bash scripts/backup-postgres.sh

restore:
	ENV_FILE=.env.production bash scripts/restore-postgres.sh $(BACKUP_FILE)

prune-backups:
	ENV_FILE=.env.production bash scripts/prune-backups.sh
