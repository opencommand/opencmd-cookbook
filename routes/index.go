package routes

import "github.com/gin-gonic/gin"

type Route interface {
	Register(router *gin.Engine)
}

func Register(router *gin.Engine, routes []Route) {
	for _, route := range routes {
		route.Register(router)
	}
}
