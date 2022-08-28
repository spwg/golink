package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
	"github.com/spwg/golink/internal/service"
)

var (
	dbPath = flag.String("db_path", "/tmp/golink.db", "Path to a sqlite database.")
	port   = flag.Int("port", 10123, "The port to listen on. Override with the PORT env var.")
)

//go:embed schema/golink.sql
var schema string

func main() {
	flag.Parse()
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	db, err := datastore.SQLite(ctx, *dbPath, schema)
	if err != nil {
		log.Fatalln(err)
	}
	gl := service.New(db)
	if os.Getenv("PORT") != "" {
		p, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return err
		}
		port = &p
	}
	return gl.Run(ctx, fmt.Sprintf(":%d", *port))
}

func init() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
}