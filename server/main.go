package main

import (
	"context"
	_ "embed"
	"flag"
	"log"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
	"github.com/spwg/golink/internal/service"
)

var (
	dbPath = flag.String("db_path", "/tmp/golink.db", "Path to a sqlite database.")
	port   = flag.Int("port", 10123, "The port to listen on.")
)

//go:embed schema/golink.sql
var schema string

func main() {
	flag.Parse()
	ctx := context.Background()
	db, err := datastore.SQLite(ctx, *dbPath, schema)
	if err != nil {
		log.Fatalln(err)
	}
	gl := service.New(db)
	if err := gl.Run(ctx, *port); err != nil {
		log.Fatalln(err)
	}
}

func init() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
}
