package routes

import (
	"opencmd-cookbook/handler"

	"github.com/gin-gonic/gin"
)

type HotfixRoute struct{}

func (r *HotfixRoute) Register(router *gin.Engine) {
	router.GET("/hotfix", handler.Hotfix)
}
