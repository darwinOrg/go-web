package test

import (
	"context"
	"fmt"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/result"
	dghttp "github.com/darwinOrg/go-httpclient"
	"github.com/darwinOrg/go-monitor"
	dgotel "github.com/darwinOrg/go-otel"
	"github.com/darwinOrg/go-web/middleware"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"testing"
	"time"
)

func TestGet(t *testing.T) {
	monitor.Start("test", 19002)
	cleanup := initTracer()
	defer cleanup()

	engine := wrapper.DefaultEngine()
	engine.Use(middleware.Tracer("test-service"))
	wrapper.Get(&wrapper.RequestHolder[wrapper.EmptyRequest, *result.Result[*UserResponse]]{
		Remark:       "测试get接口",
		RouterGroup:  engine.Group("/public"),
		RelativePath: "get",
		NonLogin:     true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *wrapper.EmptyRequest) *result.Result[*UserResponse] {
			resp := &UserResponse{
				LogUrl: "http://localhost:8080/a/b/c",
			}
			return result.Success(resp)
		},
	})
	_ = engine.Run(fmt.Sprintf(":%d", 8081))
}

func TestPost(t *testing.T) {
	monitor.Start("test", 19002)
	cleanup := initTracer()
	defer cleanup()

	engine := wrapper.DefaultEngine()
	engine.Use(middleware.Tracer("test-service"))
	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[*UserResponse]]{
		Remark:       "测试post接口",
		RouterGroup:  engine.Group("/public"),
		RelativePath: "post",
		NonLogin:     true,
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
	cleanup := initTracer()
	defer cleanup()

	engine := wrapper.DefaultEngine()
	engine.Use(middleware.Tracer("test-service"))
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

func initTracer() func() {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		panic(err)
	}

	cleanup, err := dgotel.InitTracer("test-service", exporter)
	if err != nil {
		panic(err)
	}

	dghttp.Client11 = dghttp.NewHttpClient(dghttp.NewOtelHttpTransport(dghttp.HttpTransport), 60)
	dghttp.Client11.EnableTracer = true

	return cleanup
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
