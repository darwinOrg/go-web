package wrapper

import (
	"log"

	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/gin-gonic/gin"
)

var (
	DefaultMiddlewares = []gin.HandlerFunc{middleware.Recover(), middleware.Cors(), middleware.Monitor(), middleware.Health()}
)

func init() {
	if dgsys.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
}

func DefaultEngine() *gin.Engine {
	return NewEngine(DefaultMiddlewares...)
}

func NewEngine(middlewares ...gin.HandlerFunc) *gin.Engine {
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
