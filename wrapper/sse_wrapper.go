package wrapper

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/gin-gonic/gin"
)

const sseWrittenKey = "SSE_WRITTEN"

type SseMessage struct {
	Name string `json:"name"`
	Data any    `json:"data"`
}

func SseStream(gc *gin.Context, ctx *dgctx.DgContext, messageChan chan string) {
	gc.Header("Content-Type", "text/event-stream")
	gc.Header("Cache-Control", "no-cache")
	gc.Header("Connection", "keep-alive")
	defer setSseWritten(ctx)

	for {
		select {
		case <-gc.Done():
			return
		case msg := <-messageChan:
			_, we := gc.Writer.WriteString(msg)
			if we != nil {
				dglogger.Errorf(ctx, "writing message error: %v", we)
			} else {
				dglogger.Debugf(ctx, "writing message: %s", msg)
			}
			gc.Writer.Flush()
		}
	}

}

func SendSseMessage(messageChan chan string, name string, data any) {
	msg := SseMessage{
		Name: name,
		Data: data,
	}

	messageChan <- fmt.Sprintf("data: %s\n\n", utils.MustConvertBeanToJsonString(msg))
}

func SendSseDone(messageChan chan string) {
	messageChan <- "data: DONE\n\n"
}

func setSseWritten(ctx *dgctx.DgContext) {
	ctx.SetExtraKeyValue(sseWrittenKey, true)
}

func isSseWritten(ctx *dgctx.DgContext) bool {
	if written, ok := ctx.GetExtraValue(sseWrittenKey).(bool); ok && written {
		return true
	}
	return false
}
