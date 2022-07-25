package main

import (
	"context"
	"flag"
	"log"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
	"github.com/spwg/golink/internal/service"
)

var (
	dbPath = flag.String("db_path", "/tmp/golink.db", "Path to a sqlite database.")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	db, err := datastore.SQLite(ctx, *dbPath)
	if err != nil {
		log.Fatalln(err)
	}
	gl := service.New(db)
	if err := gl.Run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func init() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
}
