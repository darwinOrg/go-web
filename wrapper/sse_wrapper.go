package wrapper

import (
	"github.com/gin-gonic/gin"
	"io"
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
