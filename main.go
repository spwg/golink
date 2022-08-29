package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3" // sql driver
	"github.com/spwg/golink/internal/datastore"
	"github.com/spwg/golink/internal/service"
)

var (
	hostName   string
	dbPathFlag = flag.String("db_path", "/tmp/golink.db", "Path to a sqlite database.")
	portFlag   = flag.Int("port", 10123, "The port to listen on. Override with the PORT env var.")
)

//go:embed internal/schema/golink.sql
var schema string

func main() {
	flag.Parse()
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalln(err)
	}
}

func run(ctx context.Context) error {
	var hostName string
	if os.Getenv("PORT") != "" {
		p, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return err
		}
		portFlag = &p
		*dbPathFlag = "/data/golink.db"
		hostName = "golinkservice.com"
	} else {
		hostName = fmt.Sprintf("localhost:%v", *portFlag)
	}
	db, err := datastore.SQLite(ctx, *dbPathFlag, schema)
	if err != nil {
		log.Fatalln(err)
	}
	gl := service.New(db, hostName)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *portFlag))
	if err != nil {
		return err
	}
	return gl.Run(ctx, l)
}

func init() {
	log.Default().SetFlags(log.LstdFlags | log.Lshortfile)
}
