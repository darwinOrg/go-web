package wrapper

import (
	"github.com/gin-gonic/gin"
	"io"
)

func SseStream(gc *gin.Context, streamFunc func() bool) {
	gc.Header("Content-Type", "text/event-stream;charset=utf-8")
	gc.Stream(func(w io.Writer) bool {
		return streamFunc()
	})
}

func SseEvent(gc *gin.Context, message any) {
	gc.SSEvent("data", message)
}

func SseDone(gc *gin.Context) {
	gc.SSEvent("data", "DONE")
}
