package wrapper

import (
	"encoding/json"
	"errors"
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
	dgsys "github.com/darwinOrg/go-common/sys"
	dglogger "github.com/darwinOrg/go-logger"
	ve "github.com/darwinOrg/go-validator-ext"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go/types"
	"net/http"
	"reflect"
	"strings"
	"time"
)

type LogLevel int

func (ll LogLevel) Value() int {
	return int(ll)
}

const (
	LOG_LEVEL_PARAM   LogLevel = 1
	LOG_LEVEL_RETURN  LogLevel = 2
	LOG_LEVEL_ALL     LogLevel = 3
	DEFAULT_LOG_LEVEL          = LOG_LEVEL_ALL
)

var myEnv = dgsys.GetProfile()

type RequestHolder[T any, V any] struct {
	*gin.RouterGroup
	RelativePath    string
	NonLogin        bool
	AllowRoles      []string
	NeedPermissions []string
	BizHandler      HandlerFunc[T, V]
	mapRequestObj   bool
	LogLevel        LogLevel
}

type MapRequest struct {
	MP map[string]any
}

type HandlerFunc[T any, V any] func(gc *gin.Context, dc *dgctx.DgContext, requestObj *T) V

func Get[T any, V any](rh *RequestHolder[T, V]) {
	rh.GET(rh.RelativePath, buildHandlerChain(rh)...)
}

func Post[T any, V any](rh *RequestHolder[T, V]) {
	rh.POST(rh.RelativePath, buildHandlerChain(rh)...)
}

func MapGet[V any](rh *RequestHolder[MapRequest, V]) {
	rh.mapRequestObj = true
	rh.GET(rh.RelativePath, buildHandlerChain(rh)...)
}

func MapPost[V any](rh *RequestHolder[MapRequest, V]) {
	rh.mapRequestObj = true
	rh.POST(rh.RelativePath, buildHandlerChain(rh)...)
}

func buildHandlerChain[T any, V any](rh *RequestHolder[T, V]) []gin.HandlerFunc {
	return []gin.HandlerFunc{loginHandler(rh), bizHandler(rh)}
}

func loginHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rh.NonLogin {
			c.Next()
			return
		}

		ctx := utils.GetDgContext(c)
		if ctx.UserId == 0 {
			dglogger.Warnf(ctx, "not login in")
			c.AbortWithStatusJSON(http.StatusOK, result.FailByError[types.Nil](dgerr.NOT_LOGIN_IN))
			return
		}

		c.Next()
	}
}

func checkEnv(c *gin.Context, ctx *dgctx.DgContext) bool {
	values := c.Request.Header["profile"]
	if len(values) == 0 || len(values[0]) == 0 {
		return true
	}
	chked := values[0] == myEnv
	if !chked {
		dglogger.Infof(ctx, "invalid profile,your profile is %s, current profile is %s", values[0], myEnv)
	}
	return chked
}

func bizHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		if rh.LogLevel.Value() == 0 {
			rh.LogLevel = DEFAULT_LOG_LEVEL
		}

		ctx := utils.GetDgContext(c)
		rp := utils.GetAllRequestParams(c, ctx)

		rpBytes, _ := json.Marshal(rp)
		dglogger.Infof(ctx, "path: %s, params: %s", c.Request.URL.Path, rpBytes)

		var rt any
		if !checkEnv(c, ctx) {
			rt = result.SimpleFail[string]("your call incorrect env")
		} else if rh.mapRequestObj {
			var ro any
			ro = &MapRequest{MP: rp}
			req := ro.(*T)
			rt = rh.BizHandler(c, ctx, req)
		} else {
			req := new(T)
			if err := c.ShouldBind(req); err != nil {
				var errs validator.ValidationErrors
				ok := errors.As(err, &errs)
				if ok {
					dglogger.Errorf(ctx, "bind request object error: %v", err)

					customErrMsgs := getCustomErrMsgs(req)

					var errMsgs []string
					for _, e := range errs {
						ns := e.Namespace()
						customErrMsg, ok2 := customErrMsgs[ns]
						if ok2 {
							errMsgs = append(errMsgs, customErrMsg)
						} else {
							translateErrMsg := getTranslateErrMsg(e, ctx.Lang)
							errMsgs = append(errMsgs, translateErrMsg)
						}
					}

					msg := strings.Join(errMsgs, "\n")

					if msg != "" {
						rt = result.SimpleFail[string](msg)
					} else {
						rt = result.FailByError[types.Nil](dgerr.ARGUMENT_NOT_VALID)
					}
				}
			} else {
				rt = rh.BizHandler(c, ctx, req)
			}
		}
		printBizHandlerLog(c, ctx, rp, rt, start, rh.LogLevel)
		c.JSON(http.StatusOK, rt)
		c.Next()
	}
}

func printBizHandlerLog(c *gin.Context, ctx *dgctx.DgContext, rp map[string]any, rt any, start time.Time, ll LogLevel) {
	ctxJson, _ := json.Marshal(ctx)
	latency := time.Now().Sub(start)
	if ll == LOG_LEVEL_ALL {
		rpBytes, _ := json.Marshal(rp)
		rtBytes, _ := json.Marshal(rt)
		dglogger.Infof(ctx, "path: %s, context: %s, params: %s, result: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, rtBytes, latency)
	} else if ll == LOG_LEVEL_PARAM {
		rpBytes, _ := json.Marshal(rp)
		dglogger.Infof(ctx, "path: %s, context: %sv, params: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, latency)
	} else if ll == LOG_LEVEL_RETURN {
		rtBytes, _ := json.Marshal(rt)
		dglogger.Infof(ctx, "path: %s, context: %s, result: %s, cost: %13v", c.Request.URL.Path, ctxJson, rtBytes, latency)
	}
}

func getCustomErrMsgs(req any) map[string]string {
	reqType := reflect.TypeOf(req)
	if reqType.Kind() != reflect.Ptr || reqType.Elem().Kind() != reflect.Struct {
		return map[string]string{}
	}

	errMsgs := map[string]string{}
	findCustomErrMsgs(reqType.Elem(), reqType.Elem().Name(), "", errMsgs)
	return errMsgs
}

func findCustomErrMsgs(tType reflect.Type, tName string, tPath string, errMsgs map[string]string) {
	var sType reflect.Type
	sTypeKind := tType.Kind()
	if sTypeKind == reflect.Ptr {
		sType = tType.Elem()
	} else {
		sType = tType
	}

	tPath = tPath + tName + "."

	for i := 0; i < sType.NumField(); i++ {
		f := sType.Field(i)
		fType := f.Type
		fName := f.Name

		errMsg := f.Tag.Get("errMsg")
		if errMsg != "" {
			ns := tPath + fName
			errMsgs[ns] = errMsg
		}

		if fType.Kind() == reflect.Ptr && fType.Elem().Kind() == reflect.Struct {
			findCustomErrMsgs(fType, fName, tPath, errMsgs)
		}
	}
}

func getTranslateErrMsg(err validator.FieldError, lng string) string {
	return ve.TranslateValidateError(validator.ValidationErrors{err}, lng)
}
