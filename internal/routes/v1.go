package routes

import (
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/middleware"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
)

func registerV1Routes(router *gin.Engine, tokenMaker token.Maker, store db.Store, handlers Handlers, opcoes Opcoes) {
	v1 := router.Group(V1Prefix)

	registerHelperRoutes(v1, handlers)

	// Rotas sem token: login, renovação e acompanhamento do cliente. São as
	// únicas em que um IP pode tentar adivinhar credenciais ou CPF/CNPJ, por
	// isso recebem limite de tentativas.
	limitadas := v1.Group("/")
	limitadas.Use(middleware.NewRateLimiter(opcoes.RateLimitRPS, opcoes.RateLimitBurst).Handler())
	registerUsuarioRoutes(limitadas, handlers)
	registerAcompanhamentoRoutes(limitadas, handlers)

	authRoutes := v1.Group("/")
	authRoutes.Use(middleware.Auth(tokenMaker, store))
	registerAuthUsuarioRoutes(authRoutes, handlers)
	registerClienteRoutes(authRoutes, handlers)
	registerVeiculoRoutes(authRoutes, handlers)
	registerServicoRoutes(authRoutes, handlers)
	registerPecaRoutes(authRoutes, handlers)
	registerOrdemServicoRoutes(authRoutes, handlers)
}

// Atalhos dos papéis usados nas rotas. O administrador passa em todas.
var (
	somenteAdmin     = middleware.RequireRole()
	atendimento      = middleware.RequireRole(middleware.PapelAtendente)
	oficina          = middleware.RequireRole(middleware.PapelMecanico)
	atendimentoEOfic = middleware.RequireRole(middleware.PapelAtendente, middleware.PapelMecanico)
)
