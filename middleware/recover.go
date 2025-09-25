package middleware

import (
	"errors"
	"net/http"

	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
	dgsys "github.com/darwinOrg/go-common/sys"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
)

type RecoverProcessor func(ctx *dgctx.DgContext, url string, params map[string]any, err error)

var recoverProcessors []RecoverProcessor

func RegisterRecoverProcessor(processor RecoverProcessor) {
	recoverProcessors = append(recoverProcessors, processor)
}

func Recover() gin.HandlerFunc {
	return gin.CustomRecovery(myRecover)
}

func myRecover(c *gin.Context, err any) {
	ctx := utils.GetDgContext(c)
	dglogger.Errorf(ctx, "panic error: %v", err)

	// 封装通用json结果返回
	c.JSON(http.StatusOK, errorToResult(c, ctx, err))
	// 终止后续接口调用，不加的话recover到异常后，还会继续执行接口里后续代码
	c.Abort()
}

func errorToResult(c *gin.Context, ctx *dgctx.DgContext, r any) any {
	switch r.(type) {
	case string:
		processRecoverError(c, ctx, errors.New(r.(string)))
		return result.SimpleFail[string](r.(string))
	case *dgerr.DgError:
		processRecoverError(c, ctx, r.(*dgerr.DgError))
		return result.FailByError[*dgerr.DgError](r.(*dgerr.DgError))
	case error:
		processRecoverError(c, ctx, r.(error))
		if dgsys.IsProd() {
			return result.SimpleFailByError(dgerr.SYSTEM_ERROR)
		} else {
			return result.SimpleFail[string](r.(error).Error())
		}
	default:
		processRecoverError(c, ctx, dgerr.SYSTEM_ERROR)
		return result.SimpleFailByError(dgerr.SYSTEM_ERROR)
	}
}

func processRecoverError(c *gin.Context, ctx *dgctx.DgContext, err error) {
	if len(recoverProcessors) > 0 {
		params := utils.GetAllRequestParams(c, ctx)
		for _, processor := range recoverProcessors {
			processor(ctx, c.Request.URL.Path, params, err)
		}
	}
}
