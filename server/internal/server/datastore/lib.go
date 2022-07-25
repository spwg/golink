// Package datastore provides persistence functionality for storing data.
package datastore

import (
	"context"
	"database/sql"
	"log"
)

// SQLite opens a *sql.DB backed by a sqlite3 database at path.
func SQLite(ctx context.Context, path string) (*sql.DB, error) {
	log.Printf("Opening database %v", path)
	return sql.Open("sqlite3", path)
}
