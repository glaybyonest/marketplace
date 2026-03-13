package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	migrationsfs "marketplace-backend/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: migrate <up|down|status|version>")
	}

	command := strings.ToLower(strings.TrimSpace(os.Args[1]))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatalf("DATABASE_URL is required")
	}

	goose.SetBaseFS(migrationsfs.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}

	db := openDB(databaseURL)
	defer db.Close()

	switch command {
	case "up":
		if err := goose.Up(db, "."); err != nil {
			log.Fatalf("goose up: %v", err)
		}
	case "down":
		if err := goose.Down(db, "."); err != nil {
			log.Fatalf("goose down: %v", err)
		}
	case "status":
		if err := goose.Status(db, "."); err != nil {
			log.Fatalf("goose status: %v", err)
		}
	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("goose version: %v", err)
		}
		fmt.Println(version)
	default:
		log.Fatalf("unsupported migrate command %q", command)
	}
}

func openDB(databaseURL string) *sql.DB {
	config, err := pgxParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("parse DATABASE_URL: %v", err)
	}

	db := stdlib.OpenDB(*config)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		log.Fatalf("ping database: %v", err)
	}
	return db
}

var pgxParseConfig = pgx.ParseConfig
