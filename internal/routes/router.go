package routes

import (
	db "github.com/Teixeiraass/backend_techchallenger/db/sqlc"
	"github.com/Teixeiraass/backend_techchallenger/token"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(tokenMaker token.Maker, store db.Store, handlers Handlers) *gin.Engine {
	router := gin.Default()

	registerSwaggerRoutes(router)
	registerV1Routes(router, tokenMaker, store, handlers)

	return router
}

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
