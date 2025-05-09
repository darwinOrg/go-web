package wrapper

import (
	"context"
	"errors"
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var DefaultMiddlewares = []gin.HandlerFunc{
	middleware.Recover(), middleware.Cors(), middleware.Monitor(), middleware.HealthHandler(),
}

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
	fillEngine(e, middlewares...)
	return e
}

func DefaultGracefulEngine(opts ...graceful.Option) *graceful.Graceful {
	return NewGracefulEngine(opts, DefaultMiddlewares...)
}

func NewGracefulEngine(opts []graceful.Option, middlewares ...gin.HandlerFunc) *graceful.Graceful {
	g, err := graceful.Default(opts...)
	if err != nil {
		panic(err)
	}
	fillEngine(g.Engine, middlewares...)
	return g
}

func RunGracefulEngine(g *graceful.Graceful) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()
	defer g.Close()

	if err := g.RunWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}

func fillEngine(e *gin.Engine, middlewares ...gin.HandlerFunc) {
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
}
