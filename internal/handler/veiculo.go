package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/gin-gonic/gin"
)

// CriarVeiculo godoc
// @Summary      Cadastrar veículo
// @Description  Cadastra um veículo para um cliente já existente. A placa é validada (padrão antigo ABC1234 ou Mercosul ABC1D23) e é única.
// @Tags         veiculos
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CriarVeiculoRequest true "Veículo"
// @Success      201 {object} dto.VeiculoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /veiculos [post]
func (server *Server) CriarVeiculo(ctx *gin.Context) {
	var req dto.CriarVeiculoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	veiculo, err := server.VeiculoService.CriarVeiculo(ctx, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, veiculo)
}

// ListarVeiculos godoc
// @Summary      Listar veículos
// @Tags         veiculos
// @Produce      json
// @Security     BearerAuth
// @Param        cliente_uuid query string false "Filtrar pelos veículos de um cliente"
// @Param        pagina       query int    false "Página (padrão 1)"
// @Param        tamanho      query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.VeiculoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /veiculos [get]
func (server *Server) ListarVeiculos(ctx *gin.Context) {
	var query dto.ListVeiculosQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	veiculos, err := server.VeiculoService.ListarVeiculos(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, veiculos)
}

// BuscarVeiculo godoc
// @Summary      Detalhar veículo
// @Tags         veiculos
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do veículo"
// @Success      200 {object} dto.VeiculoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /veiculos/{uuid} [get]
func (server *Server) BuscarVeiculo(ctx *gin.Context) {
	veiculoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	veiculo, err := server.VeiculoService.BuscarVeiculo(ctx, veiculoUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, veiculo)
}

// AtualizarVeiculo godoc
// @Summary      Atualizar veículo
// @Tags         veiculos
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                      true "UUID do veículo"
// @Param        request body dto.AtualizarVeiculoRequest true "Dados"
// @Success      200 {object} dto.VeiculoResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /veiculos/{uuid} [put]
func (server *Server) AtualizarVeiculo(ctx *gin.Context) {
	veiculoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AtualizarVeiculoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	veiculo, err := server.VeiculoService.AtualizarVeiculo(ctx, veiculoUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, veiculo)
}

// RemoverVeiculo godoc
// @Summary      Remover veículo
// @Description  Só remove veículos sem ordens de serviço.
// @Tags         veiculos
// @Security     BearerAuth
// @Param        uuid path string true "UUID do veículo"
// @Success      204
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /veiculos/{uuid} [delete]
func (server *Server) RemoverVeiculo(ctx *gin.Context) {
	veiculoUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	if err := server.VeiculoService.RemoverVeiculo(ctx, veiculoUUID); err != nil {
		handleError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
