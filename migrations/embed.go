package migrations

import "embed"

// Files contains embedded SQL migrations for the migrate CLI and production deploys.
//
//go:embed *.sql
var Files embed.FS
