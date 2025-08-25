package middleware

import (
	"net/http"

	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
)

func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.RequestURI == "/health" {
			ctx := utils.GetDgContext(c)
			dglogger.Info(ctx, "health check")
			c.AbortWithStatusJSON(http.StatusOK, "ok")
		}
	}
}
