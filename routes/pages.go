package routes

import (
	"opencmd-cookbook/handler"

	"github.com/gin-gonic/gin"
)

type PagesRoute struct{}

func (r *PagesRoute) Register(router *gin.Engine) {
	router.GET("/", handler.HomePage)
	router.POST("/explain", handler.ExplainPage)
	router.GET("/about", handler.AboutPage)
}
