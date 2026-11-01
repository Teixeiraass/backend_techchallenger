package routes

import "github.com/gin-gonic/gin"

type ClienteHandler interface {
	CriarCliente(c *gin.Context)
	ListarClientes(c *gin.Context)
	BuscarCliente(c *gin.Context)
	BuscarClientePorDocumento(c *gin.Context)
	AtualizarCliente(c *gin.Context)
	RemoverCliente(c *gin.Context)
}

func registerClienteRoutes(authRoutes gin.IRoutes, h ClienteHandler) {
	authRoutes.GET("/clientes", h.ListarClientes)
	authRoutes.GET("/clientes/:uuid", h.BuscarCliente)
	authRoutes.GET("/clientes/documento/:documento", h.BuscarClientePorDocumento)
	authRoutes.POST("/clientes", atendimento, h.CriarCliente)
	authRoutes.PUT("/clientes/:uuid", atendimento, h.AtualizarCliente)
	authRoutes.DELETE("/clientes/:uuid", atendimento, h.RemoverCliente)
}
