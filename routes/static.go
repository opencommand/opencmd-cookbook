package routes

import "github.com/gin-gonic/gin"

type StaticRoute struct{}

func (r *StaticRoute) Register(router *gin.Engine) {
	router.Static("/static", "./dist/static")
}
