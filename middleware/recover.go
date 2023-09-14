package middleware

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
	dgsys "github.com/darwinOrg/go-common/sys"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"go/types"
	"net/http"
)

func Recover() gin.HandlerFunc {
	return gin.CustomRecovery(recover)
}

func recover(c *gin.Context, err any) {
	dglogger.Errorf(&dgctx.DgContext{TraceId: utils.GetTraceId(c)}, "request error:%v", err)

	// 封装通用json结果返回
	c.JSON(http.StatusOK, errorToResult(err))
	// 终止后续接口调用，不加的话recover到异常后，还会继续执行接口里后续代码
	c.Abort()
}

func errorToResult(r any) any {
	switch r.(type) {
	case string:
		return result.SimpleFail[string](r.(string))
	case *dgerr.DgError:
		return result.FailByError[*dgerr.DgError](r.(*dgerr.DgError))
	case error:
		if dgsys.IsProd() {
			return result.FailByError[types.Nil](dgerr.SYSTEM_ERROR)
		} else {
			return result.SimpleFail[string](r.(error).Error())
		}
	default:
		return result.FailByError[types.Nil](dgerr.SYSTEM_ERROR)
	}
}
