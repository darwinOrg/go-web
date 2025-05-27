package wrapper

import (
	"bufio"
	"bytes"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dgutils "github.com/darwinOrg/go-common/utils"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	nu "net/url"
	"strings"
	"time"
)

const sseDefaultSleepTime = time.Millisecond * 10

type SseBody struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func SimpleSseStream(gc *gin.Context, messageChan chan *SseBody) {
	SseStream(gc, func(w io.Writer) bool {
		msg, ok := <-messageChan
		if ok {
			SseEvent(gc, msg.Event, msg.Data)
		} else {
			SseDone(gc)
		}
		return ok
	})
}

func SseStream(gc *gin.Context, step func(w io.Writer) bool) {
	gc.Header("Content-Type", "text/event-stream")
	gc.Header("Cache-Control", "no-cache")
	gc.Header("Connection", "keep-alive")

	gc.Stream(step)
}

func SseData(gc *gin.Context, message any) {
	gc.SSEvent("data", message)
}

func SseDone(gc *gin.Context) {
	gc.SSEvent("data", "DONE")
}

func SseEvent(gc *gin.Context, event string, message any) {
	gc.SSEvent(event, message)
}

func SseMessage(messageChan chan *SseBody, event string, message any) {
	messageChan <- &SseBody{
		Event: event,
		Data:  message,
	}
}

func SseForward(gc *gin.Context, hc *dghttp.DgHttpClient, forwardUrl string) {
	req, err := http.NewRequest(gc.Request.Method, forwardUrl, gc.Request.Body)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	req.Header = gc.Request.Header
	SseRequestRaw(gc, hc, req)
}

func SseGet(gc *gin.Context, ctx *dgctx.DgContext, hc *dghttp.DgHttpClient, url string, params map[string]string, headers map[string]string) error {
	if len(params) > 0 {
		if params != nil && len(params) > 0 {
			vs := nu.Values{}
			for k, v := range params {
				vs.Add(k, v)
			}
			url += dgutils.IfReturn(strings.Contains(url, "?"), "&", "?")
			url += vs.Encode()
		}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		dglogger.Errorf(ctx, "new sse get request error, url: %s, err: %v", url, err)
		return err
	}

	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Set(key, val)
		}
	}

	SseRequestRaw(gc, hc, req)
	return nil
}

func SsePostJson(gc *gin.Context, ctx *dgctx.DgContext, hc *dghttp.DgHttpClient, url string, params any, headers map[string]string) error {
	paramsBytes, err := dglogger.Json(params)
	if err != nil {
		dglogger.Errorf(ctx, "json marshal error, url: %s, params: %v, err: %v", url, params, err)
		return err
	}
	dglogger.Infof(ctx, "sse post request, url: %s, params: %v", url, string(paramsBytes))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(paramsBytes))
	if err != nil {
		dglogger.Errorf(ctx, "new sse request error, url: %s, params: %v, err: %v", url, params, err)
		return err
	}

	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Set(key, val)
		}
	}

	SseRequestRaw(gc, hc, req)
	return nil
}

func SseRequestRaw(gc *gin.Context, hc *dghttp.DgHttpClient, req *http.Request) {
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	ctx := utils.GetDgContext(gc)

	resp, err := hc.DoRequestRaw(ctx, req)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}
	defer func() { _ = resp.Body.Close() }()

	statusCode := utils.AdapterStatusCode(resp.StatusCode)
	gc.Status(statusCode)
	utils.WriteHeaders(gc, resp.Header)

	reader := bufio.NewReader(resp.Body)
	for {
		rawLine, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF {
			break
		}

		if len(rawLine) > 0 {
			_, _ = gc.Writer.Write(rawLine)
			gc.Writer.Flush()
		}

		time.Sleep(sseDefaultSleepTime)
	}
}
