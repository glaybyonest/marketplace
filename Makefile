SHELL := /bin/bash

GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.26.0
DB_URL ?= $(DATABASE_URL)
TEST_DB_URL ?= $(TEST_DATABASE_URL)

.PHONY: up down build run migrate-up migrate-down seed test lint fmt ci frontend-install frontend-dev openapi-check

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
