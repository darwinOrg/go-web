package wrapper

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/gin-gonic/gin"
)

type SseMessage struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

func SseStream(gc *gin.Context, ctx *dgctx.DgContext, messageChan chan string) {
	gc.Header("Content-Type", "text/event-stream;charset=utf-8")
	gc.Header("Cache-Control", "no-cache")
	gc.Header("Connection", "keep-alive")

	for msg := range messageChan {
		_, we := gc.Writer.WriteString(msg)
		if we != nil {
			dglogger.Errorf(ctx, "writing message error: %v", we)
		} else {
			dglogger.Debugf(ctx, "writing message: %s", msg)
		}
		gc.Writer.Flush()
	}
}

func SendSseMessage(messageChan chan string, name string, data any) {
	msg := SseMessage{
		Name: name,
		Data: data,
	}

	messageChan <- fmt.Sprintf("data: %s\n\n", utils.MustConvertBeanToJsonString(msg))
}
