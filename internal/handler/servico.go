package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/gin-gonic/gin"
)

// CriarServico godoc
// @Summary      Cadastrar serviço no catálogo
// @Description  Somente administrador. Preço em centavos.
// @Tags         servicos
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CriarServicoRequest true "Serviço"
// @Success      201 {object} dto.ServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /servicos [post]
func (server *Server) CriarServico(ctx *gin.Context) {
	var req dto.CriarServicoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	servico, err := server.ServicoService.CriarServico(ctx, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewServicoResponse(readmodel.DeServico(*servico)))
}

// ListarServicos godoc
// @Summary      Listar serviços do catálogo
// @Tags         servicos
// @Produce      json
// @Security     BearerAuth
// @Param        busca         query string false "Filtro por nome"
// @Param        apenas_ativos query bool   false "Somente serviços ativos"
// @Param        pagina        query int    false "Página (padrão 1)"
// @Param        tamanho       query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.ServicoResponse
// @Failure      400 {object} ErrorResponse
// @Router       /servicos [get]
func (server *Server) ListarServicos(ctx *gin.Context) {
	var query dto.ListServicosQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	servicos, err := server.ServicoService.ListarServicos(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.ServicoResponse, 0, len(servicos))
	for _, s := range servicos {
		response = append(response, dto.NewServicoResponse(readmodel.DeServico(s)))
	}
	ctx.JSON(http.StatusOK, response)
}

// BuscarServico godoc
// @Summary      Detalhar serviço
// @Tags         servicos
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do serviço"
// @Success      200 {object} dto.ServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /servicos/{uuid} [get]
func (server *Server) BuscarServico(ctx *gin.Context) {
	servicoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	servico, err := server.ServicoService.BuscarServico(ctx, servicoUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewServicoResponse(readmodel.DeServico(*servico)))
}

// AtualizarServico godoc
// @Summary      Atualizar serviço
// @Description  Somente administrador. O novo preço vale para itens incluídos depois; OS já orçadas mantêm o valor da época.
// @Tags         servicos
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                      true "UUID do serviço"
// @Param        request body dto.AtualizarServicoRequest true "Dados"
// @Success      200 {object} dto.ServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /servicos/{uuid} [put]
func (server *Server) AtualizarServico(ctx *gin.Context) {
	servicoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AtualizarServicoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	servico, err := server.ServicoService.AtualizarServico(ctx, servicoUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewServicoResponse(readmodel.DeServico(*servico)))
}

// InativarServico godoc
// @Summary      Inativar serviço
// @Description  Somente administrador. O serviço sai do catálogo (não entra em OS nova), mas o histórico das OS é mantido.
// @Tags         servicos
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do serviço"
// @Success      200 {object} dto.ServicoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /servicos/{uuid} [delete]
func (server *Server) InativarServico(ctx *gin.Context) {
	servicoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	servico, err := server.ServicoService.InativarServico(ctx, servicoUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewServicoResponse(readmodel.DeServico(*servico)))
}
