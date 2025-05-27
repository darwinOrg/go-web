package wrapper

import (
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func HttpForward(gc *gin.Context, ctx *dgctx.DgContext, hc *dghttp.DgHttpClient, forwardUrl string) {
	req, err := http.NewRequest(gc.Request.Method, forwardUrl, gc.Request.Body)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	req.Header = gc.Request.Header
	dglogger.Debugf(ctx, "raw request header: %v", req.Header)

	statusCode, headers, body, err := hc.DoRequest(ctx, req)
	dglogger.Debugf(ctx, "forward url[%s], statusCode:%d, body:%s", forwardUrl, statusCode, body)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	statusCode = utils.AdapterStatusCode(statusCode)
	gc.Status(statusCode)
	utils.WriteHeaders(gc, headers)

	if len(body) > 0 {
		_, _ = gc.Writer.Write(body)
	} else {
		_, _ = gc.Writer.Write([]byte{})
	}
}
