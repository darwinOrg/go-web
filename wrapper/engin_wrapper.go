package wrapper

import (
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/gin-gonic/gin"
)

func DefaultEngine() *gin.Engine {
	setGinMod()
	e := gin.New()
	e.UseH2C = true
	e.Use(middleware.Recover(), middleware.Cors(), middleware.Monitor())
	return e
}

func NewEngine(middlewares ...gin.HandlerFunc) *gin.Engine {
	setGinMod()
	e := gin.New()
	e.UseH2C = true
	e.Use(middlewares...)

	return e
}

func setGinMod() {
	if dgsys.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
}
