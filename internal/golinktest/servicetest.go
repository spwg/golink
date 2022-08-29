// Package golinktest provides functionality for testing the service.
package golinktest

import (
	"context"
	"database/sql"
	_ "embed"
	"net"
	"path"
	"testing"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
)

//go:generate cp -r ../schema ./schema
//go:embed schema/golink.sql
var schema string

// NewDatabase creates a database.
func NewDatabase(ctx context.Context, t *testing.T) *sql.DB {
	t.Helper()
	dbPath := path.Join(t.TempDir(), "db.sql")
	db, err := datastore.SQLite(ctx, dbPath, schema)
	if err != nil {
		t.Fatalf("SQLite(%q) failed: %v", dbPath, err)
	}
	return db
}

// Listen starts a listener.
func Listen(ctx context.Context, t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen(%q, %q) failed: %v", "tcp", ":0", err)
	}
	return l
}

type runner interface {
	Run(ctx context.Context, listener net.Listener) error
}

// RunServer runs the service.
func RunServer(ctx context.Context, t *testing.T, svc runner, l net.Listener) {
	t.Helper()
	if err := svc.Run(ctx, l); err != nil {
		t.Errorf("Run(%q) failed: %v", l, err)
	}
}
