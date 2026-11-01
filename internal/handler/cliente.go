package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/gin-gonic/gin"
)

// CriarCliente godoc
// @Summary      Cadastrar cliente
// @Description  Cadastra um cliente identificado por CPF ou CNPJ (validado pelos dígitos verificadores e único).
// @Tags         clientes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CriarClienteRequest true "Cliente"
// @Success      201 {object} dto.ClienteResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /clientes [post]
func (server *Server) CriarCliente(ctx *gin.Context) {
	var req dto.CriarClienteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	cliente, err := server.ClienteService.CriarCliente(ctx, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewClienteResponse(*cliente))
}

// ListarClientes godoc
// @Summary      Listar clientes
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        busca   query string false "Filtro por nome ou documento"
// @Param        pagina  query int    false "Página (padrão 1)"
// @Param        tamanho query int    false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.ClienteResponse
// @Failure      400 {object} ErrorResponse
// @Router       /clientes [get]
func (server *Server) ListarClientes(ctx *gin.Context) {
	var query dto.ListClientesQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	clientes, err := server.ClienteService.ListarClientes(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.ClienteResponse, 0, len(clientes))
	for _, c := range clientes {
		response = append(response, dto.NewClienteResponse(c))
	}
	ctx.JSON(http.StatusOK, response)
}

// BuscarCliente godoc
// @Summary      Detalhar cliente
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        uuid path string true "UUID do cliente"
// @Success      200 {object} dto.ClienteResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /clientes/{uuid} [get]
func (server *Server) BuscarCliente(ctx *gin.Context) {
	clienteUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	cliente, err := server.ClienteService.BuscarCliente(ctx, clienteUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewClienteResponse(*cliente))
}

// BuscarClientePorDocumento godoc
// @Summary      Identificar cliente por CPF/CNPJ
// @Tags         clientes
// @Produce      json
// @Security     BearerAuth
// @Param        documento path string true "CPF ou CNPJ (somente números)"
// @Success      200 {object} dto.ClienteResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /clientes/documento/{documento} [get]
func (server *Server) BuscarClientePorDocumento(ctx *gin.Context) {
	cliente, err := server.ClienteService.BuscarClientePorDocumento(ctx, ctx.Param("documento"))
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewClienteResponse(*cliente))
}

// AtualizarCliente godoc
// @Summary      Atualizar cliente
// @Description  Atualiza nome e contato. O CPF/CNPJ identifica o cliente e não muda.
// @Tags         clientes
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                      true "UUID do cliente"
// @Param        request body dto.AtualizarClienteRequest true "Dados"
// @Success      200 {object} dto.ClienteResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /clientes/{uuid} [put]
func (server *Server) AtualizarCliente(ctx *gin.Context) {
	clienteUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AtualizarClienteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	cliente, err := server.ClienteService.AtualizarCliente(ctx, clienteUUID, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewClienteResponse(*cliente))
}

// RemoverCliente godoc
// @Summary      Remover cliente
// @Description  Só remove clientes sem veículos e sem ordens de serviço.
// @Tags         clientes
// @Security     BearerAuth
// @Param        uuid path string true "UUID do cliente"
// @Success      204
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /clientes/{uuid} [delete]
func (server *Server) RemoverCliente(ctx *gin.Context) {
	clienteUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	if err := server.ClienteService.RemoverCliente(ctx, clienteUUID); err != nil {
		handleError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
