package wrapper

import (
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/gin-gonic/gin"
)

func DefaultEngine() *gin.Engine {
	return NewEngine(middleware.Recover(), middleware.Cors(), middleware.Monitor())
}

func NewEngine(middlewares ...gin.HandlerFunc) *gin.Engine {
	if dgsys.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	e := gin.New()
	e.UseH2C = true
	e.MaxMultipartMemory = 8 << 20
	e.Use(middlewares...)

	return e
}
