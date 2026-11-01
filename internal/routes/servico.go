package routes

import "github.com/gin-gonic/gin"

type ServicoHandler interface {
	CriarServico(c *gin.Context)
	ListarServicos(c *gin.Context)
	BuscarServico(c *gin.Context)
	AtualizarServico(c *gin.Context)
	InativarServico(c *gin.Context)
}

func registerServicoRoutes(authRoutes gin.IRoutes, h ServicoHandler) {
	authRoutes.GET("/servicos", h.ListarServicos)
	authRoutes.GET("/servicos/:uuid", h.BuscarServico)
	authRoutes.POST("/servicos", somenteAdmin, h.CriarServico)
	authRoutes.PUT("/servicos/:uuid", somenteAdmin, h.AtualizarServico)
	authRoutes.DELETE("/servicos/:uuid", somenteAdmin, h.InativarServico)
}
