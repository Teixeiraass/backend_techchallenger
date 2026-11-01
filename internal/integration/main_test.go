// Package integration exercises the main flows through the real HTTP router
// and a real PostgreSQL database (the one configured in app.env/DB_SOURCE,
// with the migrations applied). Skipped with -short or without a database.
package integration

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"testing"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/handler"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var testRouter *gin.Engine

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() || os.Getenv("SKIP_DB_TESTS") == "true" {
		log.Println("skipping integration tests")
		os.Exit(0)
	}

	gin.SetMode(gin.TestMode)

	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	if config.TokenSymmetricKey == "" {
		config.TokenSymmetricKey = util.RandomString(32)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}
	if err := conn.Ping(); err != nil {
		log.Println("skipping integration tests, database not reachable:", err)
		os.Exit(0)
	}

	server, err := handler.NewServer(config, db.NewStore(conn), notification.NewNoopSender())
	if err != nil {
		log.Fatal("cannot create server:", err)
	}
	testRouter = server.Router()

	os.Exit(m.Run())
}
