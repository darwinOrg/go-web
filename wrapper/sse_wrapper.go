package wrapper

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/gin-gonic/gin"
)

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

func SseMessage(messageChan chan string, message string) {
	messageChan <- fmt.Sprintf("data: %s\n\n", message)
}
