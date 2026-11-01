package routes

import "github.com/gin-gonic/gin"

type UsuarioHandler interface {
	Login(c *gin.Context)
	RenewAccessToken(c *gin.Context)
	Logout(c *gin.Context)
	GetUsuarioAtual(c *gin.Context)
	CriarUsuario(c *gin.Context)
	ListarUsuarios(c *gin.Context)
	AtualizarUsuarioAtivo(c *gin.Context)
}

func registerUsuarioRoutes(router gin.IRoutes, h UsuarioHandler) {
	router.POST("/usuarios/login", h.Login)
	router.POST("/tokens/refresh", h.RenewAccessToken)
}

func registerAuthUsuarioRoutes(authRoutes gin.IRoutes, h UsuarioHandler) {
	authRoutes.GET("/usuarios/me", h.GetUsuarioAtual)
	authRoutes.POST("/usuarios/logout", h.Logout)
	authRoutes.POST("/usuarios", somenteAdmin, h.CriarUsuario)
	authRoutes.GET("/usuarios", somenteAdmin, h.ListarUsuarios)
	authRoutes.PATCH("/usuarios/:uuid/ativo", somenteAdmin, h.AtualizarUsuarioAtivo)
}
