package middleware

import (
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func HealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.RequestURI == "/health" {
			ctx := utils.GetDgContext(c)
			dglogger.Info(ctx, "health check")
			c.AbortWithStatusJSON(http.StatusOK, "ok")
		}
	}
}
