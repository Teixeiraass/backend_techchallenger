package handler

import (
	"fmt"

	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/eventos"
	"github.com/Teixeiraass/backend_techchallenger/internal/repository"
	"github.com/Teixeiraass/backend_techchallenger/internal/routes"
	"github.com/Teixeiraass/backend_techchallenger/internal/service"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/Teixeiraass/backend_techchallenger/util"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config              util.Config
	store               db.Store
	tokenMaker          token.Maker
	router              *gin.Engine
	uow                 service.UnidadeDeTrabalho
	despachante         *eventos.Despachante
	UsuarioService      service.UsuarioService
	ClienteService      service.ClienteService
	VeiculoService      service.VeiculoService
	ServicoService      service.ServicoService
	PecaService         service.PecaService
	OrdemServicoService service.OrdemServicoService
}

// NewServer monta as dependências: a unidade de trabalho e os repositórios
// (portas do domínio), o despachante de eventos (as políticas são assinadas
// em cmd/server/main.go) e os serviços.
func NewServer(config util.Config, store db.Store, despachante *eventos.Despachante) (*Server, error) {
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
		PecaService:         service.NewPecaService(uow, store, despachante),
		OrdemServicoService: service.NewOrdemServicoService(uow, store, despachante),
		uow:                 uow,
		despachante:         despachante,
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

// Catalogo expõe a leitura de cadastro para as políticas de notificação.
func (server *Server) Catalogo() service.Catalogo {
	return server.uow.Leitura().Catalogo()
}
