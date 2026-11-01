package routes

import "github.com/gin-gonic/gin"

type HelperHandler interface {
	GetHelper(c *gin.Context)
}

func registerHelperRoutes(router gin.IRoutes, h HelperHandler) {
	router.GET("/helper", h.GetHelper)
}
