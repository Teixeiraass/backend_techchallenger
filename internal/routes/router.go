package routes

import (
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/internal/middleware"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Tamanho máximo do corpo de qualquer requisição (1 MiB).
const limiteCorpoBytes = 1 << 20

// Opcoes ajusta as proteções do roteador.
type Opcoes struct {
	// RateLimitRPS e RateLimitBurst limitam, por IP, o login, a renovação de
	// token e as rotas públicas de acompanhamento.
	RateLimitRPS   float64
	RateLimitBurst int
}

func Setup(tokenMaker token.Maker, store db.Store, handlers Handlers, opcoes Opcoes) *gin.Engine {
	// gin.New em vez de gin.Default: o logger padrão grava a URL inteira,
	// com a query string, e a consulta pública pode levar CPF/CNPJ.
	router := gin.New()
	router.Use(gin.Recovery(), middleware.Logger(), middleware.LimiteCorpo(limiteCorpoBytes))

	registerSwaggerRoutes(router)
	registerV1Routes(router, tokenMaker, store, handlers, opcoes)

	return router
}

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
