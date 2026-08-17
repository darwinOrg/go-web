package wrapper

import (
	"net/http"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	"github.com/gin-gonic/gin"
)

func HttpForward(c *gin.Context, ctx *dgctx.DgContext, hc *dghttp.DgHttpClient, forwardUrl string) {
	request, err := dghttp.CopyRequest(ctx, c.Request, forwardUrl, c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	resp, err := hc.DoRequestRaw(ctx, request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	WriteResponse(c, ctx, resp)
}

func WriteResponse(c *gin.Context, ctx *dgctx.DgContext, response *http.Response) {
	statusCode, headers, body, err := dghttp.ExtractResponse(ctx, response)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	statusCode = adapterStatusCode(statusCode)
	c.Status(statusCode)
	writeHeaders(c, headers)

	if len(body) > 0 {
		_, _ = c.Writer.Write(body)
	} else {
		_, _ = c.Writer.Write([]byte{})
	}
}

func writeHeaders(c *gin.Context, headers map[string][]string) {
	for k, v := range headers {
		if len(v) == 0 || v[0] == "" {
			c.Writer.Header().Del(k)
			continue
		}
		c.Writer.Header()[k] = v
	}
}

func adapterStatusCode(code int) int {
	if code >= http.StatusInternalServerError {
		return http.StatusInternalServerError
	}

	return code
}
