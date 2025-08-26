package middleware

import (
	"time"

	"github.com/darwinOrg/go-common/utils"
	"github.com/darwinOrg/go-monitor"
	"github.com/gin-gonic/gin"
)

const (
	falseString = "false"
	trueString  = "true"
)

func Monitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		monitor.HttpServerCounter(path)
		start := time.Now()
		c.Next()
		monitor.HttpServerDuration(path, utils.IfReturn(len(c.Errors) > 0, trueString, falseString), time.Since(start).Milliseconds())
	}
}
