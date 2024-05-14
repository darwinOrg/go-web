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

func DefaultEngine() *gin.Engine {
	return NewEngine(middleware.Recover(), middleware.Cors(), middleware.Monitor(), middleware.HealthHandler())
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

	return e
}

func GracefulRun(e *gin.Engine, addr string) {
	g, err := graceful.New(e, graceful.WithAddr(addr))
	if err != nil {
		panic(err)
	}
	defer g.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	if err = g.RunWithContext(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Print(err)
	}
}
