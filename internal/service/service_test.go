package service

import (
	"context"
	"database/sql"
	_ "embed"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spwg/golink/internal/golinktest"
	"github.com/spwg/golink/internal/link"
)

func TestIndex(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	db := golinktest.NewDatabase(ctx, t)
	l := golinktest.Listen(ctx, t)
	go golinktest.RunServer(ctx, t, New(db), l)
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
	db := golinktest.NewDatabase(ctx, t)
	addEntry(ctx, t, db, "foo", "http://example.com")
	l := golinktest.Listen(ctx, t)
	go golinktest.RunServer(ctx, t, New(db), l)
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
