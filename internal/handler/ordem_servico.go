package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/gin-gonic/gin"
)

// AbrirOrdemServico godoc
// @Summary      Abrir ordem de serviço
// @Description  Identifica o cliente pelo CPF/CNPJ e o veículo pela placa (que precisa pertencer ao cliente). Serviços e peças são opcionais na abertura; o orçamento é calculado automaticamente. A OS nasce com status RECEBIDA.
// @Tags         ordens-servico
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.AbrirOrdemServicoRequest true "Dados da OS"
// @Success      201 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico [post]
func (server *Server) AbrirOrdemServico(ctx *gin.Context) {
	var req dto.AbrirOrdemServicoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.AbrirOrdemServico(ctx, req, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, os)
}

// ListarOrdensServico godoc
// @Summary      Listar ordens de serviço
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        status       query string false "RECEBIDA, EM_DIAGNOSTICO, AGUARDANDO_APROVACAO, EM_EXECUCAO, FINALIZADA ou ENTREGUE"
// @Param        cliente_uuid query string false "Filtrar por cliente"
// @Param        pagina       query int    false "Página (padrão 1)"
// @Param        tamanho      query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.OrdemServicoResumoResponse
// @Failure      400 {object} ErrorResponse
// @Router       /ordens-servico [get]
func (server *Server) ListarOrdensServico(ctx *gin.Context) {
	var query dto.ListOrdensServicoQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	ordens, err := server.OrdemServicoService.ListarOrdensServico(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ordens)
}

// BuscarOrdemServico godoc
// @Summary      Detalhar ordem de serviço
// @Description  Itens, orçamento, reparos adicionais, histórico de status e tempos.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da OS"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /ordens-servico/{uuid} [get]
func (server *Server) BuscarOrdemServico(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.BuscarOrdemServico(ctx, osUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// AdicionarServicoOS godoc
// @Summary      Incluir serviço na OS
// @Description  Só serviço ativo do catálogo. Permitido com a OS Recebida ou Em diagnóstico. O orçamento é recalculado.
// @Tags         ordens-servico
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                 true "UUID da OS"
// @Param        request body dto.ItemServicoRequest true "Serviço"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/servicos [post]
func (server *Server) AdicionarServicoOS(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.ItemServicoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.AdicionarServico(ctx, osUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// RemoverServicoOS godoc
// @Summary      Remover serviço da OS
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid      path string true "UUID da OS"
// @Param        item_uuid path string true "UUID do item de serviço"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/servicos/{item_uuid} [delete]
func (server *Server) RemoverServicoOS(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}
	itemUUID, ok := uuidParam(ctx, "item_uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.RemoverServico(ctx, osUUID, itemUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// AdicionarPecaOS godoc
// @Summary      Incluir peça ou insumo na OS
// @Description  Só entra se houver estoque (considerando o que a OS já pediu da mesma peça). Permitido com a OS Recebida ou Em diagnóstico. O orçamento é recalculado.
// @Tags         ordens-servico
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string              true "UUID da OS"
// @Param        request body dto.ItemPecaRequest true "Peça"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/pecas [post]
func (server *Server) AdicionarPecaOS(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.ItemPecaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.AdicionarPeca(ctx, osUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// RemoverPecaOS godoc
// @Summary      Remover peça da OS
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid      path string true "UUID da OS"
// @Param        item_uuid path string true "UUID do item de peça"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/pecas/{item_uuid} [delete]
func (server *Server) RemoverPecaOS(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}
	itemUUID, ok := uuidParam(ctx, "item_uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.RemoverPeca(ctx, osUUID, itemUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// IniciarDiagnostico godoc
// @Summary      Iniciar diagnóstico
// @Description  Recebida -> Em diagnóstico. Mecânico ou administrador.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da OS"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/diagnostico [post]
func (server *Server) IniciarDiagnostico(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.IniciarDiagnostico(ctx, osUUID, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// EnviarOrcamento godoc
// @Summary      Enviar orçamento ao cliente
// @Description  Em diagnóstico -> Aguardando aprovação. Mecânico ou administrador. Recebe o texto do diagnóstico (opcional), recalcula o orçamento, notifica o cliente por e-mail e disponibiliza a aprovação pelo link de acompanhamento.
// @Tags         ordens-servico
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                     true  "UUID da OS"
// @Param        request body dto.EnviarOrcamentoRequest false "Diagnóstico"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/orcamento/enviar [post]
func (server *Server) EnviarOrcamento(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.EnviarOrcamentoRequest
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			bindError(ctx, err)
			return
		}
	}

	os, err := server.OrdemServicoService.EnviarOrcamento(ctx, osUUID, req, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// IniciarExecucao godoc
// @Summary      Iniciar execução
// @Description  Aguardando aprovação -> Em execução, com orçamento aprovado. Baixa o estoque na mesma transação. Normalmente é disparado automaticamente pela aprovação; esta rota serve para tentar de novo depois de repor peças em falta.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da OS"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/execucao [post]
func (server *Server) IniciarExecucao(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.IniciarExecucao(ctx, osUUID, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// FinalizarOrdemServico godoc
// @Summary      Finalizar OS
// @Description  Em execução -> Finalizada. Registra o tempo de execução e notifica o cliente. Bloqueado se houver reparo adicional pendente.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da OS"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/finalizar [post]
func (server *Server) FinalizarOrdemServico(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.FinalizarOrdemServico(ctx, osUUID, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// EntregarVeiculo godoc
// @Summary      Entregar veículo
// @Description  Finalizada -> Entregue. Também entrega o veículo sem execução (Em diagnóstico -> Entregue) quando o orçamento foi rejeitado.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da OS"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/entregar [post]
func (server *Server) EntregarVeiculo(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	os, err := server.OrdemServicoService.EntregarVeiculo(ctx, osUUID, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// RegistrarReparoAdicional godoc
// @Summary      Registrar reparo adicional
// @Description  Com a OS Em execução, o mecânico registra um problema novo com serviços e peças. Os itens só entram no orçamento depois da autorização do cliente.
// @Tags         ordens-servico
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                              true "UUID da OS"
// @Param        request body dto.RegistrarReparoAdicionalRequest true "Reparo"
// @Success      201 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/reparos-adicionais [post]
func (server *Server) RegistrarReparoAdicional(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.RegistrarReparoAdicionalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.RegistrarReparoAdicional(ctx, osUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, os)
}

// TempoMedioExecucao godoc
// @Summary      Tempo médio de execução
// @Description  Média entre o início da execução e a finalização das OS finalizadas, no geral e por serviço. Filtro opcional pela data de finalização.
// @Tags         ordens-servico
// @Produce      json
// @Security     BearerAuth
// @Param        data_inicio query string false "AAAA-MM-DD"
// @Param        data_fim    query string false "AAAA-MM-DD (inclusiva)"
// @Success      200 {object} dto.TempoMedioExecucaoResponse
// @Failure      400 {object} ErrorResponse
// @Router       /ordens-servico/tempo-medio [get]
func (server *Server) TempoMedioExecucao(ctx *gin.Context) {
	var query dto.TempoMedioQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	rsp, err := server.OrdemServicoService.TempoMedioExecucao(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rsp)
}
