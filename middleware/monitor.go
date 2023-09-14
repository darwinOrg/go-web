package middleware

import (
	"github.com/darwinOrg/go-monitor"
	"github.com/gin-gonic/gin"
	"time"
)

func Monitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		monitor.HttpServerCounter(path)
		start := time.Now().UnixMilli()
		c.Next()

		cost := time.Now().UnixMilli() - start
		e := "false"
		if len(c.Errors) > 0 {
			e = "true"
		}
		monitor.HttpServerDuration(path, e, cost)
	}
}
