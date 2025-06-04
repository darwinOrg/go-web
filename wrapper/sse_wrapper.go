package wrapper

import (
	"bufio"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"time"
)

const sseDefaultSleepTime = time.Millisecond * 10

var DefaultSseHttpClient = dghttp.NewHttpClient(dghttp.Http2Transport, 24*60*60)

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

func SseForward(gc *gin.Context, ctx *dgctx.DgContext, forwardUrl string) {
	request, err := http.NewRequest(gc.Request.Method, forwardUrl, gc.Request.Body)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}
	request.Header = gc.Request.Header
	dghttp.WriteSseHeaders(request)

	resp, err := DefaultSseHttpClient.DoRequestRaw(ctx, request)
	if err != nil {
		gc.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	WriteSseResponse(gc, resp)
}

func SseGet(gc *gin.Context, ctx *dgctx.DgContext, url string, params map[string]string, headers map[string]string) error {
	resp, err := DefaultSseHttpClient.SseGet(ctx, url, params, headers)
	if err != nil {
		return err
	}

	WriteSseResponse(gc, resp)
	return nil
}

func SsePostJson(gc *gin.Context, ctx *dgctx.DgContext, url string, params any, headers map[string]string) error {
	resp, err := DefaultSseHttpClient.SsePostJson(ctx, url, params, headers)
	if err != nil {
		return err
	}

	WriteSseResponse(gc, resp)
	return nil
}

func WriteSseResponse(gc *gin.Context, resp *http.Response) {
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
