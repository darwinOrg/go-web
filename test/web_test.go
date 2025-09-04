package test

import (
	"fmt"

	"testing"
	"time"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-monitor"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
)

func TestGet(t *testing.T) {
	monitor.Start("test", 19002)
	engine := wrapper.DefaultEngine()
	wrapper.RegisterSlowThresholdProcessor(func(ctx *dgctx.DgContext, url string, timeout, cost time.Duration) {
		dglogger.Warnf(ctx, "请求超时, url: %s, timeout: %v, cost: %v", url, timeout, cost)
	})
	wrapper.Get(&wrapper.RequestHolder[wrapper.EmptyRequest, *result.Result[*UserResponse]]{
		Remark:        "测试get接口",
		RouterGroup:   engine.Group("/public"),
		RelativePath:  "get",
		NonLogin:      true,
		EnableTracer:  true,
		SlowThreshold: time.Second,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *wrapper.EmptyRequest) *result.Result[*UserResponse] {
			resp := &UserResponse{
				LogUrl: "http://localhost:8080/a/b/c",
			}
			return result.Success(resp)
		},
	})
	_ = engine.Run(fmt.Sprintf(":%d", 8080))
}

func TestPost(t *testing.T) {
	monitor.Start("test", 19002)
	engine := wrapper.DefaultEngine()
	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*UserResponse]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/public"),
		RelativePath: "post",
		NonLogin:     true,
		EnableTracer: true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *UserRequest) *result.Result[*UserResponse] {
			_, _ = dghttp.Client11.DoGet(ctx, "https://www.baidu.com",
				map[string]string{
					"param1": "param1 value",
					"param2": "param2 value",
				},
				map[string]string{
					"header1": "header1 value",
					"header2": "header2 value",
				})

			resp := &UserResponse{
				LogUrl: "http://localhost:8080/a/b/c",
			}

			return result.Success(resp)
		},
	})
	_ = engine.Run(fmt.Sprintf(":%d", 8080))
}

func TestSSE(t *testing.T) {
	monitor.Start("test", 19002)
	engine := wrapper.DefaultEngine()
	wrapper.Get(&wrapper.RequestHolder[result.Void, *result.Result[*result.Void]]{
		Remark:       "测试sse接口",
		RouterGroup:  engine.Group("/public"),
		RelativePath: "sse",
		NonLogin:     true,
		EnableTracer: true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *result.Void) *result.Result[*result.Void] {
			handleSSE(gc)
			return result.SimpleSuccess()
		},
	})
	_ = engine.Run(fmt.Sprintf(":%d", 8080))
}

func handleSSE(c *gin.Context) {
	messageChan := make(chan *wrapper.SseBody)

	go func() {
		defer close(messageChan)
		for i := 0; i < 5; i++ {
			messageChan <- &wrapper.SseBody{Event: "data", Data: i}
			time.Sleep(time.Second)
		}
	}()

	wrapper.SimpleSseStream(c, messageChan, true)
}

type UserRequest struct {
	Name     string    `json:"name" errMsg:"姓名错误:不能为空" remark:"名称"`
	Age      int       `json:"age" remark:"年龄"`
	UserInfo *userInfo `json:"userInfo"`
}

type UserResponse struct {
	LogUrl string `json:"logUrl" appendUid:"true"`
}

type userInfo struct {
	Sex int `errMsg:"性别错误" remark:"性别"`
}
