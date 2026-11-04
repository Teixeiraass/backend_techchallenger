package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/routes"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/notification"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config              util.Config
	store               db.Store
	tokenMaker          token.Maker
	router              *gin.Engine
	UsuarioService      service.UsuarioService
	ClienteService      service.ClienteService
	VeiculoService      service.VeiculoService
	ServicoService      service.ServicoService
	PecaService         service.PecaService
	OrdemServicoService service.OrdemServicoService
}

func NewServer(config util.Config, store db.Store, sender notification.Sender) (*Server, error) {
	tokenMaker, err := token.NewMaker(config.TokenType, config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:              config,
		store:               store,
		tokenMaker:          tokenMaker,
		UsuarioService:      service.NewUsuarioService(store, tokenMaker, config),
		ClienteService:      service.NewClienteService(store),
		VeiculoService:      service.NewVeiculoService(store),
		ServicoService:      service.NewServicoService(store),
		PecaService:         service.NewPecaService(store, sender, config),
		OrdemServicoService: service.NewOrdemServicoService(store, sender, config),
	}

	// Sem configuração explícita vale o padrão de 1 req/s com rajada de 10.
	if config.RateLimitRPS <= 0 {
		config.RateLimitRPS = 1
	}
	if config.RateLimitBurst <= 0 {
		config.RateLimitBurst = 10
	}
	server.router = routes.Setup(server.tokenMaker, server.store, server, routes.Opcoes{
		RateLimitRPS:   config.RateLimitRPS,
		RateLimitBurst: config.RateLimitBurst,
	})

	return server, nil
}

// Start sobe o servidor HTTP com timeouts (contra conexões lentas, Slowloris)
// e encerra de forma graciosa quando ctx é cancelado.
func (server *Server) Start(ctx context.Context, address string) error {
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	erros := make(chan error, 1)
	go func() {
		erros <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-erros:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

// Router exposes the gin engine (used by the integration tests).
func (server *Server) Router() *gin.Engine {
	return server.router
}
