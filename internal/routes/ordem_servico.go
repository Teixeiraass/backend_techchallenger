package routes

import "github.com/gin-gonic/gin"

type OrdemServicoHandler interface {
	AbrirOrdemServico(c *gin.Context)
	ListarOrdensServico(c *gin.Context)
	BuscarOrdemServico(c *gin.Context)
	AdicionarServicoOS(c *gin.Context)
	RemoverServicoOS(c *gin.Context)
	AdicionarPecaOS(c *gin.Context)
	RemoverPecaOS(c *gin.Context)
	IniciarDiagnostico(c *gin.Context)
	EnviarOrcamento(c *gin.Context)
	IniciarExecucao(c *gin.Context)
	FinalizarOrdemServico(c *gin.Context)
	EntregarVeiculo(c *gin.Context)
	RegistrarReparoAdicional(c *gin.Context)
	TempoMedioExecucao(c *gin.Context)
}

func registerOrdemServicoRoutes(authRoutes gin.IRoutes, h OrdemServicoHandler) {
	authRoutes.GET("/ordens-servico", h.ListarOrdensServico)
	authRoutes.GET("/ordens-servico/tempo-medio", somenteAdmin, h.TempoMedioExecucao)
	authRoutes.GET("/ordens-servico/:uuid", h.BuscarOrdemServico)

	authRoutes.POST("/ordens-servico", atendimento, h.AbrirOrdemServico)
	authRoutes.POST("/ordens-servico/:uuid/servicos", atendimentoEOfic, h.AdicionarServicoOS)
	authRoutes.DELETE("/ordens-servico/:uuid/servicos/:item_uuid", atendimentoEOfic, h.RemoverServicoOS)
	authRoutes.POST("/ordens-servico/:uuid/pecas", atendimentoEOfic, h.AdicionarPecaOS)
	authRoutes.DELETE("/ordens-servico/:uuid/pecas/:item_uuid", atendimentoEOfic, h.RemoverPecaOS)

	authRoutes.POST("/ordens-servico/:uuid/diagnostico", oficina, h.IniciarDiagnostico)
	authRoutes.POST("/ordens-servico/:uuid/orcamento/enviar", oficina, h.EnviarOrcamento)
	authRoutes.POST("/ordens-servico/:uuid/execucao", atendimentoEOfic, h.IniciarExecucao)
	authRoutes.POST("/ordens-servico/:uuid/reparos-adicionais", oficina, h.RegistrarReparoAdicional)
	authRoutes.POST("/ordens-servico/:uuid/finalizar", oficina, h.FinalizarOrdemServico)
	authRoutes.POST("/ordens-servico/:uuid/entregar", atendimento, h.EntregarVeiculo)
}
