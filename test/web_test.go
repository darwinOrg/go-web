package test

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-monitor"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	monitor.Start("test", 19002)

	engine := wrapper.DefaultEngine()
	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[*UserResponse]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/public"),
		RelativePath: "get",
		NonLogin:     true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *wrapper.MapRequest) *result.Result[*UserResponse] {
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
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *UserRequest) *result.Result[*UserResponse] {
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
	Name     string    `json:"name" binding:"required" errMsg:"姓名错误:不能为空" remark:"名称"`
	Age      int       `json:"age" binding:"required,gt=0,lt=100" remark:"年龄"`
	UserInfo *userInfo `json:"userInfo" binding:"required"`
}

type UserResponse struct {
	LogUrl string `json:"logUrl" appendUid:"true"`
}

type userInfo struct {
	Sex int `binding:"required,gt=0,lt=5" errMsg:"性别错误" remark:"性别"`
}
