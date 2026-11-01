package routes

import "github.com/gin-gonic/gin"

// AcompanhamentoHandler agrupa as rotas públicas do cliente (sem token).
type AcompanhamentoHandler interface {
	ConsultarAndamento(c *gin.Context)
	AprovarOrcamento(c *gin.Context)
	RejeitarOrcamento(c *gin.Context)
	AutorizarReparoAdicional(c *gin.Context)
	RecusarReparoAdicional(c *gin.Context)
}

func registerAcompanhamentoRoutes(router gin.IRoutes, h AcompanhamentoHandler) {
	router.GET("/ordens-servico/:uuid/andamento", h.ConsultarAndamento)
	router.POST("/ordens-servico/:uuid/orcamento/aprovar", h.AprovarOrcamento)
	router.POST("/ordens-servico/:uuid/orcamento/rejeitar", h.RejeitarOrcamento)
	router.POST("/ordens-servico/:uuid/reparos-adicionais/:reparo_uuid/autorizar", h.AutorizarReparoAdicional)
	router.POST("/ordens-servico/:uuid/reparos-adicionais/:reparo_uuid/recusar", h.RecusarReparoAdicional)
}
