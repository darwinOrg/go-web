package wrapper

import (
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func HttpForward(c *gin.Context, hc *dghttp.DgHttpClient, forwardUrl string) {
	req, err := http.NewRequest(c.Request.Method, forwardUrl, c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	req.Header = c.Request.Header
	ctx := utils.GetDgContext(c)
	dglogger.Debugf(ctx, "raw request header: %v", req.Header)

	statusCode, headers, body, err := hc.DoRequest(ctx, req)
	dglogger.Debugf(ctx, "forward url[%s], statusCode:%d, body:%s", forwardUrl, statusCode, body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	statusCode = utils.AdapterStatusCode(statusCode)
	c.Status(statusCode)
	utils.WriteHeaders(c, headers)

	if len(body) > 0 {
		_, _ = c.Writer.Write(body)
	} else {
		_, _ = c.Writer.Write([]byte{})
	}
}
