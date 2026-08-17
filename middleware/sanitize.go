package middleware

import (
	"bytes"
	"io"

	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
)

var ugcPolicy = bluemonday.UGCPolicy()

func SanitizeBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.IsPost(c) {
			raw := utils.GetBodyBytes(c)
			clean := ugcPolicy.SanitizeBytes(raw)
			c.Request.Body = io.NopCloser(bytes.NewReader(clean))
			c.Request.ContentLength = int64(len(clean))
		}
		c.Next()
	}
}
