package routes

import "github.com/gin-gonic/gin"

type VeiculoHandler interface {
	CriarVeiculo(c *gin.Context)
	ListarVeiculos(c *gin.Context)
	BuscarVeiculo(c *gin.Context)
	AtualizarVeiculo(c *gin.Context)
	RemoverVeiculo(c *gin.Context)
}

func registerVeiculoRoutes(authRoutes gin.IRoutes, h VeiculoHandler) {
	authRoutes.GET("/veiculos", h.ListarVeiculos)
	authRoutes.GET("/veiculos/:uuid", h.BuscarVeiculo)
	authRoutes.POST("/veiculos", atendimento, h.CriarVeiculo)
	authRoutes.PUT("/veiculos/:uuid", atendimento, h.AtualizarVeiculo)
	authRoutes.DELETE("/veiculos/:uuid", atendimento, h.RemoverVeiculo)
}
