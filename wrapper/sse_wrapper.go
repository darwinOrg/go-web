package wrapper

import (
	"bufio"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

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

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		if len(bytes) > 0 {
			_, _ = gc.Writer.Write(bytes)
			gc.Writer.Flush()
		}
	}
}
