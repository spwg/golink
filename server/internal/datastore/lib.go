// Package datastore provides persistence functionality for storing data.
package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
)

// SQLite opens a *sql.DB backed by a sqlite3 database at path.
func SQLite(ctx context.Context, path string) (*sql.DB, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot create database handle: %w", err)
	}
	log.Printf("Opening database %v", path)
	return sql.Open("sqlite3", path)
}
