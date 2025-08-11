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
		start := time.Now()
		c.Next()

		e := "false"
		if len(c.Errors) > 0 {
			e = "true"
		}
		monitor.HttpServerDuration(path, e, time.Since(start).Milliseconds())
	}
}
