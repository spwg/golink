// Package datastore provides persistence functionality for storing data.
package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
)

// SQLite opens a *sql.DB backed by a sqlite3 database at path.
func SQLite(ctx context.Context, path, schema string) (*sql.DB, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("cannot create database handle: %w", err)
		}
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("failed to close the new database: %w", err)
		}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to execute schema statements: %w", err)
	}
	return db, nil
}
