package handler

import (
	"net/http"

	"github.com/Teixeiraass/backend_techchallenger/internal/dto"
	"github.com/Teixeiraass/backend_techchallenger/internal/readmodel"
	"github.com/gin-gonic/gin"
)

// Login godoc
// @Summary      Login
// @Description  Autentica um usuário da oficina (administrador, atendente ou mecânico) e retorna o access token e o refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Credenciais"
// @Success      200 {object} dto.LoginResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /usuarios/login [post]
func (server *Server) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	rsp, err := server.UsuarioService.Login(ctx, req, ctx.Request.UserAgent(), ctx.ClientIP())
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rsp)
}

// RenewAccessToken godoc
// @Summary      Renovar token de acesso
// @Description  Gera um novo access token a partir de um refresh token válido.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RenewAccessTokenRequest true "Refresh token"
// @Success      200 {object} dto.RenewAccessTokenResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /tokens/refresh [post]
func (server *Server) RenewAccessToken(ctx *gin.Context) {
	var req dto.RenewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	rsp, err := server.UsuarioService.RenewAccessToken(ctx, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, rsp)
}

// Logout godoc
// @Summary      Logout
// @Description  Bloqueia a sessão (refresh token) do usuário autenticado.
// @Tags         auth
// @Accept       json
// @Security     BearerAuth
// @Param        request body dto.LogoutRequest true "Sessão"
// @Success      204
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /usuarios/logout [post]
func (server *Server) Logout(ctx *gin.Context) {
	var req dto.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	if err := server.UsuarioService.Logout(ctx, req.SessionID, currentUserID(ctx)); err != nil {
		handleError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// GetUsuarioAtual godoc
// @Summary      Usuário autenticado
// @Tags         usuarios
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.UsuarioResponse
// @Failure      401 {object} ErrorResponse
// @Router       /usuarios/me [get]
func (server *Server) GetUsuarioAtual(ctx *gin.Context) {
	payload := currentPayload(ctx)
	usuario, err := server.UsuarioService.BuscarUsuario(ctx, payload.UserUUID)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewUsuarioResponse(readmodel.DeUsuario(*usuario)))
}

// CriarUsuario godoc
// @Summary      Criar usuário da oficina
// @Description  Somente administrador. Papéis: ADMINISTRADOR, ATENDENTE ou MECANICO.
// @Tags         usuarios
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CriarUsuarioRequest true "Usuário"
// @Success      201 {object} dto.UsuarioResponse
// @Failure      400 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /usuarios [post]
func (server *Server) CriarUsuario(ctx *gin.Context) {
	var req dto.CriarUsuarioRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	usuario, err := server.UsuarioService.CriarUsuario(ctx, req)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, dto.NewUsuarioResponse(readmodel.DeUsuario(*usuario)))
}

// ListarUsuarios godoc
// @Summary      Listar usuários da oficina
// @Description  Somente administrador.
// @Tags         usuarios
// @Produce      json
// @Security     BearerAuth
// @Param        pagina  query int false "Página (padrão 1)"
// @Param        tamanho query int false "Itens por página (padrão 20, máx 100)"
// @Success      200 {array} dto.UsuarioResponse
// @Failure      403 {object} ErrorResponse
// @Router       /usuarios [get]
func (server *Server) ListarUsuarios(ctx *gin.Context) {
	var query dto.PaginacaoQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		bindError(ctx, err)
		return
	}

	usuarios, err := server.UsuarioService.ListarUsuarios(ctx, query)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response := make([]dto.UsuarioResponse, 0, len(usuarios))
	for _, u := range usuarios {
		response = append(response, dto.NewUsuarioResponse(readmodel.DeUsuario(u)))
	}
	ctx.JSON(http.StatusOK, response)
}

// AtualizarUsuarioAtivo godoc
// @Summary      Ativar ou desativar usuário
// @Description  Somente administrador. Usuário inativo não consegue mais acessar a API.
// @Tags         usuarios
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        uuid    path string                           true "UUID do usuário"
// @Param        request body dto.AtualizarUsuarioAtivoRequest true "Situação"
// @Success      200 {object} dto.UsuarioResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /usuarios/{uuid}/ativo [patch]
func (server *Server) AtualizarUsuarioAtivo(ctx *gin.Context) {
	usuarioUUID, ok := uuidParam(ctx, "uuid")
	if !ok {
		return
	}

	var req dto.AtualizarUsuarioAtivoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		bindError(ctx, err)
		return
	}

	usuario, err := server.UsuarioService.AtualizarAtivo(ctx, usuarioUUID, *req.Ativo)
	if err != nil {
		handleError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.NewUsuarioResponse(readmodel.DeUsuario(*usuario)))
}
