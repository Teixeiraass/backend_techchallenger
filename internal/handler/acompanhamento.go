package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/gin-gonic/gin"
)

// Rotas públicas usadas pelo cliente (link de acompanhamento ou aplicativo).
// Não exigem token: o UUID da OS + o CPF/CNPJ do dono funcionam como
// credencial. Documento errado devolve 404, igual a OS inexistente.

// ConsultarAndamento godoc
// @Summary      Consultar andamento da OS (cliente)
// @Description  Rota pública. O cliente informa o CPF/CNPJ do dono da OS para acompanhar status, orçamento, reparos adicionais e histórico.
// @Tags         acompanhamento
// @Produce      json
// @Param        uuid      path  string true "UUID da OS"
// @Param        documento query string true "CPF ou CNPJ do cliente"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/andamento [get]
func (server *Server) ConsultarAndamento(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var query dto.AndamentoQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.ConsultarAndamento(ctx, osUUID, query.Documento)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// AprovarOrcamento godoc
// @Summary      Aprovar orçamento (cliente)
// @Description  Rota pública. Registra a aprovação e o sistema inicia a execução automaticamente, baixando o estoque. Se faltar peça, a aprovação fica registrada e o campo "aviso" explica que a execução aguarda reposição.
// @Tags         acompanhamento
// @Accept       json
// @Produce      json
// @Param        uuid    path string                    true "UUID da OS"
// @Param        request body dto.DecisaoClienteRequest true "Documento do cliente"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/orcamento/aprovar [post]
func (server *Server) AprovarOrcamento(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.DecisaoClienteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.AprovarOrcamento(ctx, osUUID, req.Documento)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// RejeitarOrcamento godoc
// @Summary      Rejeitar orçamento (cliente)
// @Description  Rota pública. A OS volta para Em diagnóstico: a oficina pode montar um novo orçamento ou devolver o veículo.
// @Tags         acompanhamento
// @Accept       json
// @Produce      json
// @Param        uuid    path string                    true "UUID da OS"
// @Param        request body dto.DecisaoClienteRequest true "Documento do cliente"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/orcamento/rejeitar [post]
func (server *Server) RejeitarOrcamento(ctx *gin.Context) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.DecisaoClienteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	os, err := server.OrdemServicoService.RejeitarOrcamento(ctx, osUUID, req.Documento)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}

// AutorizarReparoAdicional godoc
// @Summary      Autorizar reparo adicional (cliente)
// @Description  Rota pública. Os itens do reparo entram no orçamento e as peças são baixadas do estoque.
// @Tags         acompanhamento
// @Accept       json
// @Produce      json
// @Param        uuid        path string                    true "UUID da OS"
// @Param        reparo_uuid path string                    true "UUID do reparo adicional"
// @Param        request     body dto.DecisaoClienteRequest true "Documento do cliente"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/reparos-adicionais/{reparo_uuid}/autorizar [post]
func (server *Server) AutorizarReparoAdicional(ctx *gin.Context) {
	server.decidirReparoAdicional(ctx, true)
}

// RecusarReparoAdicional godoc
// @Summary      Recusar reparo adicional (cliente)
// @Description  Rota pública. O reparo não entra na OS.
// @Tags         acompanhamento
// @Accept       json
// @Produce      json
// @Param        uuid        path string                    true "UUID da OS"
// @Param        reparo_uuid path string                    true "UUID do reparo adicional"
// @Param        request     body dto.DecisaoClienteRequest true "Documento do cliente"
// @Success      200 {object} dto.OrdemServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /ordens-servico/{uuid}/reparos-adicionais/{reparo_uuid}/recusar [post]
func (server *Server) RecusarReparoAdicional(ctx *gin.Context) {
	server.decidirReparoAdicional(ctx, false)
}

func (server *Server) decidirReparoAdicional(ctx *gin.Context, autorizar bool) {
	osUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}
	reparoUUID, ok := uuidParam(ctx, "reparo_uuid")
	if !ok {
		return
	}

	var req dto.DecisaoClienteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	var (
		os  *dto.OrdemServicoResponse
		err error
	)
	if autorizar {
		os, err = server.OrdemServicoService.AutorizarReparoAdicional(ctx, osUUID, reparoUUID, req.Documento)
	} else {
		os, err = server.OrdemServicoService.RecusarReparoAdicional(ctx, osUUID, reparoUUID, req.Documento)
	}
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, os)
}
