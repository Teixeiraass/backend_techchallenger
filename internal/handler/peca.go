package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/gin-gonic/gin"
)

// CriarPeca godoc
// @Summary      Cadastrar peça ou insumo
// @Description  Somente administrador. O saldo inicial gera um movimento de ENTRADA.
// @Tags         pecas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CriarPecaRequest true "Peça ou insumo"
// @Success      201 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /pecas [post]
func (server *Server) CriarPeca(ctx *gin.Context) {
	var req dto.CriarPecaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	peca, err := server.PecaService.CriarPeca(ctx, req, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewPecaResponse(*peca))
}

// ListarPecas godoc
// @Summary      Listar peças e insumos
// @Tags         pecas
// @Produce      json
// @Security     BearerAuth
// @Param        busca         query string false "Filtro por nome ou código"
// @Param        tipo          query string false "PECA ou INSUMO"
// @Param        apenas_ativas query bool   false "Somente ativas"
// @Param        pagina        query int    false "Página (padrão 1)"
// @Param        tamanho       query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Router       /pecas [get]
func (server *Server) ListarPecas(ctx *gin.Context) {
	var query dto.ListPecasQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	pecas, err := server.PecaService.ListarPecas(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.PecaResponse, 0, len(pecas))
	for _, p := range pecas {
		response = append(response, dto.NewPecaResponse(p))
	}
	ctx.JSON(http.StatusOK, response)
}

// ListarEstoqueBaixo godoc
// @Summary      Peças no estoque mínimo
// @Description  Peças e insumos ativos com saldo menor ou igual ao estoque mínimo (precisam de reposição).
// @Tags         pecas
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.PecaResponse
// @Router       /pecas/estoque-baixo [get]
func (server *Server) ListarEstoqueBaixo(ctx *gin.Context) {
	pecas, err := server.PecaService.ListarEstoqueBaixo(ctx)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.PecaResponse, 0, len(pecas))
	for _, p := range pecas {
		response = append(response, dto.NewPecaResponse(p))
	}
	ctx.JSON(http.StatusOK, response)
}

// BuscarPeca godoc
// @Summary      Detalhar peça ou insumo
// @Tags         pecas
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da peça"
// @Success      200 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /pecas/{uuid} [get]
func (server *Server) BuscarPeca(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	peca, err := server.PecaService.BuscarPeca(ctx, pecaUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewPecaResponse(*peca))
}

// AtualizarPeca godoc
// @Summary      Atualizar peça ou insumo
// @Description  Somente administrador. O saldo só muda por reposição, ajuste ou baixa.
// @Tags         pecas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                   true "UUID da peça"
// @Param        request body dto.AtualizarPecaRequest true "Dados"
// @Success      200 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /pecas/{uuid} [put]
func (server *Server) AtualizarPeca(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AtualizarPecaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	peca, err := server.PecaService.AtualizarPeca(ctx, pecaUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewPecaResponse(*peca))
}

// InativarPeca godoc
// @Summary      Inativar peça ou insumo
// @Description  Somente administrador. Peça inativa não entra em OS nova.
// @Tags         pecas
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID da peça"
// @Success      200 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /pecas/{uuid} [delete]
func (server *Server) InativarPeca(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	peca, err := server.PecaService.InativarPeca(ctx, pecaUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewPecaResponse(*peca))
}

// ReporEstoque godoc
// @Summary      Repor estoque
// @Description  Somente administrador. Entrada de unidades (quantidade maior que zero).
// @Tags         pecas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                  true "UUID da peça"
// @Param        request body dto.ReporEstoqueRequest true "Reposição"
// @Success      200 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /pecas/{uuid}/reposicao [post]
func (server *Server) ReporEstoque(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.ReporEstoqueRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	peca, err := server.PecaService.ReporEstoque(ctx, pecaUUID, req, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewPecaResponse(*peca))
}

// AjustarEstoque godoc
// @Summary      Ajustar estoque (inventário)
// @Description  Somente administrador. Define o saldo contado no inventário; a diferença vira um movimento de AJUSTE.
// @Tags         pecas
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                    true "UUID da peça"
// @Param        request body dto.AjustarEstoqueRequest true "Ajuste"
// @Success      200 {object} dto.PecaResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /pecas/{uuid}/ajuste [post]
func (server *Server) AjustarEstoque(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AjustarEstoqueRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	peca, err := server.PecaService.AjustarEstoque(ctx, pecaUUID, req, currentUserID(ctx))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewPecaResponse(*peca))
}

// ListarMovimentosEstoque godoc
// @Summary      Movimentos de estoque da peça
// @Description  Histórico de entradas, saídas (baixas por OS) e ajustes.
// @Tags         pecas
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path  string true  "UUID da peça"
// @Param        pagina  query int    false "Página (padrão 1)"
// @Param        tamanho query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.MovimentoEstoqueResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /pecas/{uuid}/movimentos [get]
func (server *Server) ListarMovimentosEstoque(ctx *gin.Context) {
	pecaUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var query dto.PaginacaoQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	movimentos, err := server.PecaService.ListarMovimentos(ctx, pecaUUID, query)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.MovimentoEstoqueResponse, 0, len(movimentos))
	for _, m := range movimentos {
		response = append(response, dto.NewMovimentoEstoqueResponse(readmodel.DeMovimentoEstoque(m)))
	}
	ctx.JSON(http.StatusOK, response)
}
