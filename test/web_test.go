package test

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
	"github.com/darwinOrg/go-monitor"
	"github.com/darwinOrg/go-web/wrapper"
	"github.com/gin-gonic/gin"
	"go/types"
	"testing"
)

func TestGet(t *testing.T) {
	monitor.Start("test", 19002)

	engine := wrapper.DefaultEngine()
	wrapper.Get(&wrapper.RequestHolder[wrapper.MapRequest, *result.Result[types.Nil]]{
		RouterGroup:  engine.Group("/test"),
		RelativePath: "/get",
		NonLogin:     true,
		BizHandler: func(_ *gin.Context, ctx *dgctx.DgContext, request *wrapper.MapRequest) *result.Result[types.Nil] {
			return result.FailByError[types.Nil](dgerr.ARGUMENT_NOT_VALID)
		},
	})
	engine.Run(fmt.Sprintf(":%d", 8080))
}

func TestPost(t *testing.T) {
	monitor.Start("test", 19002)

	engine := wrapper.DefaultEngine()
	wrapper.Post(&wrapper.RequestHolder[UserRequest, *result.Result[string]]{
		RouterGroup:  engine.Group("/test"),
		RelativePath: "post",
		NonLogin:     true,
		BizHandler: func(gc *gin.Context, ctx *dgctx.DgContext, request *UserRequest) *result.Result[string] {
			return result.Success("Success")
		},
	})
	engine.Run(fmt.Sprintf(":%d", 8080))
}

type UserRequest struct {
	Name     string    `binding:"required" errMsg:"姓名错误:不能为空"`
	Age      int       `binding:"required,gt=0,lt=100"`
	UserInfo *userInfo `binding:"required"`
}

type userInfo struct {
	Sex int `binding:"required,gt=0,lt=5" errMsg:"性别错误"`
}
