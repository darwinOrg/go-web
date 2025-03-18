package wrapper

import (
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/gin-gonic/gin"
	"log"
)

func DefaultEngine() *gin.Engine {
	return NewEngine(middleware.Recover(), middleware.Cors(), middleware.Monitor(), middleware.HealthHandler(), middleware.CopyBody())
}

func NewEngine(middlewares ...gin.HandlerFunc) *gin.Engine {
	if dgsys.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	e := gin.New()
	e.UseH2C = true
	e.MaxMultipartMemory = 8 << 20
	e.Use(middlewares...)
	_ = e.SetTrustedProxies(nil)
	e.HandleMethodNotAllowed = true
	e.NoRoute(func(c *gin.Context) {
		log.Printf("404 Not Found: uri: %s, method: %s", c.Request.URL.Path, c.Request.Method)
	})
	e.NoMethod(func(c *gin.Context) {
		log.Printf("405 Method Not Allowed: uri: %s, method: %s", c.Request.URL.Path, c.Request.Method)
	})

	return e
}
