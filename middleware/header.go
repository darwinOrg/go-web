package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
)

// NoCache is a middleware function that appends headers
// to prevent the client from caching the HTTP response.
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Next()
	}
}

// Options is a middleware function that appends headers
// for options requests and aborts then exits the middleware
// chain and ends the request.
func Options() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "OPTIONS" {
			c.Next()
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
			c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
			c.Header("Content-Type", "application/json")
			c.AbortWithStatus(200)
		}
	}
}

// Secure is a middleware function that appends security
// and resource access headers.
func Secure() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		//c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000")
		}

		// Also consider adding Content-Security-Policy headers
		// c.Header("Content-Security-Policy", "script-src 'self' https://cdnjs.cloudflare.com")
	}
}

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
