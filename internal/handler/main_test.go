package handler

import (
	"os"
	"testing"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store, notification.NewNoopSender())
	require.NoError(t, err)

	return server
}

// JWT é a única forma de autenticação: basta a chave, sem escolher o tipo.
func TestNewServerUsaJWT(t *testing.T) {
	server := newTestServer(t, nil)
	require.IsType(t, &token.JWTMaker{}, server.tokenMaker)

	_, err := NewServer(util.Config{TokenSymmetricKey: util.RandomString(31)}, nil, notification.NewNoopSender())
	require.Error(t, err)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Exit(m.Run())
}
