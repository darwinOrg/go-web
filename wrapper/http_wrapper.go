package wrapper

import (
	"net/http"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	dgotel "github.com/darwinOrg/go-otel"
	"github.com/gin-gonic/gin"
)

func HttpForward(gc *gin.Context, ctx *dgctx.DgContext, hc *dghttp.DgHttpClient, forwardUrl string) {
	var (
		request *http.Request
		err     error
	)
	if DefaultSseHttpClient.EnableTracer && ctx.GetInnerContext() != nil {
		dgotel.SetSpanAttributesByDgContext(ctx)
		request, err = http.NewRequestWithContext(ctx.GetInnerContext(), gc.Request.Method, forwardUrl, gc.Request.Body)
	} else {
		request, err = http.NewRequest(gc.Request.Method, forwardUrl, gc.Request.Body)
	}
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	request.Header = gc.Request.Header
	resp, err := hc.DoRequestRaw(ctx, request)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	WriteResponse(gc, ctx, resp)
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
	} else {
		return code
	}
}
