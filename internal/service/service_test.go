package service

import (
	"context"
	"database/sql"
	_ "embed"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
	"github.com/spwg/golink/internal/link"
)

//go:generate cp -r ../schema ./schema
//go:embed schema/golink.sql
var schema string

func newDatabase(ctx context.Context, t *testing.T) *sql.DB {
	t.Helper()
	dbPath := path.Join(t.TempDir(), "db.sql")
	db, err := datastore.SQLite(ctx, dbPath, schema)
	if err != nil {
		t.Fatalf("SQLite(%q) failed: %v", dbPath, err)
	}
	return db
}

func listen(ctx context.Context, t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen(%q, %q) failed: %v", "tcp", ":0", err)
	}
	return l
}

func runServer(ctx context.Context, t *testing.T, service *GoLink, l net.Listener) {
	t.Helper()
	if err := service.Run(ctx, l); err != nil {
		t.Errorf("Run(%q) failed: %v", l, err)
	}
}

func TestIndex(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	db := newDatabase(ctx, t)
	l := listen(ctx, t)
	go runServer(ctx, t, New(db), l)
	time.Sleep(500 * time.Millisecond)
	url := "http://" + l.Addr().String()
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Get(%q) returned err=%v, want nil", url, err)
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("Get(%q) returned code=%v, want %v", url, got, want)
	}
}

func addEntry(ctx context.Context, t *testing.T, db *sql.DB, name, address string) {
	t.Helper()
	if err := link.Create(ctx, db, name, address); err != nil {
		t.Fatalf("Create(%q, %q) failed: %v", name, address, err)
	}
}

func TestGoLinkPage(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	db := newDatabase(ctx, t)
	addEntry(ctx, t, db, "foo", "http://example.com")
	l := listen(ctx, t)
	go runServer(ctx, t, New(db), l)
	time.Sleep(500 * time.Millisecond)
	url := "http://" + l.Addr().String() + "/golink/foo"
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Get(%q) returned err=%v, want nil", url, err)
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("Get(%q) returned code=%v, want %v", url, got, want)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "foo") {
		t.Errorf("Get(%q) returned page without %v, want the page to contain %q:\n%s", url, "foo", "foo", s)
	}
	if !strings.Contains(s, "http://example.com") {
		t.Errorf("Get(%q) returned page without %v, want the page to contain %q:\n%s", url, "http://example.com", "http://example.com", s)
	}
}
