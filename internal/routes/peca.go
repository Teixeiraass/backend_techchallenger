package routes

import "github.com/gin-gonic/gin"

type PecaHandler interface {
	CriarPeca(c *gin.Context)
	ListarPecas(c *gin.Context)
	ListarEstoqueBaixo(c *gin.Context)
	BuscarPeca(c *gin.Context)
	AtualizarPeca(c *gin.Context)
	InativarPeca(c *gin.Context)
	ReporEstoque(c *gin.Context)
	AjustarEstoque(c *gin.Context)
	ListarMovimentosEstoque(c *gin.Context)
}

func registerPecaRoutes(authRoutes gin.IRoutes, h PecaHandler) {
	authRoutes.GET("/pecas", h.ListarPecas)
	authRoutes.GET("/pecas/estoque-baixo", h.ListarEstoqueBaixo)
	authRoutes.GET("/pecas/:uuid", h.BuscarPeca)
	authRoutes.GET("/pecas/:uuid/movimentos", h.ListarMovimentosEstoque)
	authRoutes.POST("/pecas", somenteAdmin, h.CriarPeca)
	authRoutes.PUT("/pecas/:uuid", somenteAdmin, h.AtualizarPeca)
	authRoutes.DELETE("/pecas/:uuid", somenteAdmin, h.InativarPeca)
	authRoutes.POST("/pecas/:uuid/reposicao", somenteAdmin, h.ReporEstoque)
	authRoutes.POST("/pecas/:uuid/ajuste", somenteAdmin, h.AjustarEstoque)
}
