package middleware

import (
	"github.com/gin-gonic/gin"
	"strings"
)

func CorrectHeaderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		newHeader := make(map[string][]string)

		for k, v := range c.Request.Header {
			var exists bool
			for _, ah := range AllowHeaders {
				if strings.EqualFold(k, ah) {
					exists = true
					newHeader[ah] = v
					break
				}
			}

			if !exists {
				newHeader[k] = v
			}
		}

		c.Request.Header = newHeader
	}
}
