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
