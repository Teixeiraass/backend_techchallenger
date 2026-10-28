package db

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/Teixeiraass/backend_techchallenger/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB
var testStore Store

// TestMain connects to the database configured in app.env / DB_SOURCE.
// The tests are skipped with -short or when the database is not reachable,
// so `go test -short ./...` works without Docker.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() || os.Getenv("SKIP_DB_TESTS") == "true" {
		log.Println("skipping db tests")
		os.Exit(0)
	}

	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	if err := testDB.Ping(); err != nil {
		log.Println("skipping db tests, database not reachable:", err)
		os.Exit(0)
	}

	testQueries = New(testDB)
	testStore = NewStore(testDB)

	os.Exit(m.Run())
}
