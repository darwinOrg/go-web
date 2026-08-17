package wrapper

import (
	"bufio"
	"io"
	"net/http"
	"time"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	"github.com/gin-gonic/gin"
)

const sseDefaultSleepTime = time.Millisecond * 10

var DefaultSseHttpClient = dghttp.NewHttpClient(dghttp.Http2Transport, 24*60*60)

type SseBody struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func SimpleSseStream(c *gin.Context, messageChan chan *SseBody, sendDoneEvent bool) {
	SseStream(c, func(w io.Writer) bool {
		msg, ok := <-messageChan
		if ok {
			SseEvent(c, msg.Event, msg.Data)
		} else if sendDoneEvent {
			SseDone(c)
		}
		return ok
	})
}

func SseStream(c *gin.Context, step func(w io.Writer) bool) {
	c.Header("Content-Type", "text/event-stream;charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(step)
}

func SseData(c *gin.Context, message any) {
	c.SSEvent("data", message)
}

func SseDone(c *gin.Context) {
	c.SSEvent("data", "DONE")
}

func SseEvent(c *gin.Context, event string, message any) {
	c.SSEvent(event, message)
}

func SseMessage(messageChan chan *SseBody, event string, message any) {
	messageChan <- &SseBody{
		Event: event,
		Data:  message,
	}
}

func SseForward(c *gin.Context, ctx *dgctx.DgContext, forwardUrl string) {
	request, err := dghttp.CopyRequest(ctx, c.Request, forwardUrl, c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	dghttp.WriteSseHeaders(request)

	resp, err := DefaultSseHttpClient.DoRequestRaw(ctx, request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusOK, result.SimpleFailByError(err))
		return
	}

	WriteSseResponse(c, resp)
}

func SseGet(c *gin.Context, ctx *dgctx.DgContext, url string, params map[string]string, headers map[string]string) error {
	resp, err := DefaultSseHttpClient.SseGet(ctx, url, params, headers)
	if err != nil {
		return err
	}

	WriteSseResponse(c, resp)
	return nil
}

func SsePostJson(c *gin.Context, ctx *dgctx.DgContext, url string, params any, headers map[string]string) error {
	resp, err := DefaultSseHttpClient.SsePostJson(ctx, url, params, headers)
	if err != nil {
		return err
	}

	WriteSseResponse(c, resp)
	return nil
}

func WriteSseResponse(c *gin.Context, resp *http.Response) {
	defer func() { _ = resp.Body.Close() }()

	statusCode := adapterStatusCode(resp.StatusCode)
	c.Status(statusCode)
	writeHeaders(c, resp.Header)
	reader := bufio.NewReader(resp.Body)

	for {
		rawLine, readErr := reader.ReadBytes('\n')
		if readErr == io.EOF {
			break
		}

		if len(rawLine) > 0 {
			_, _ = c.Writer.Write(rawLine)
			c.Writer.Flush()
		}

		time.Sleep(sseDefaultSleepTime)
	}
}
