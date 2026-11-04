package handler

import (
	"fmt"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
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

	// Os comandos da OS e do estoque passam pela unidade de trabalho e pelos
	// repositórios (portas do domínio); as listagens leem o Store direto.
	uow := repository.NewUnidadeDeTrabalho(store)

	server := &Server{
		config:              config,
		store:               store,
		tokenMaker:          tokenMaker,
		UsuarioService:      service.NewUsuarioService(store, tokenMaker, config),
		ClienteService:      service.NewClienteService(store),
		VeiculoService:      service.NewVeiculoService(store),
		ServicoService:      service.NewServicoService(store),
		PecaService:         service.NewPecaService(uow, store, sender, config),
		OrdemServicoService: service.NewOrdemServicoService(uow, store, sender, config),
	}

	server.router = routes.Setup(server.tokenMaker, server.store, server)

	return server, nil
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// Router exposes the gin engine (used by the integration tests).
func (server *Server) Router() *gin.Engine {
	return server.router
}
