package wrapper

import (
	"encoding/json"
	"errors"
	dgcoll "github.com/darwinOrg/go-common/collection"
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
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
		if rh.mapRequestObj {
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

					errNss := dgcoll.Trans2Map(errs, func(fe validator.FieldError) string { return fe.Namespace() })

					customErrMsg, otherErrs := getCustomErrMsg(req, errNss)
					translatedErrMsg := getTranslateErrMsg(otherErrs, ctx.Lang)

					errMsg := strings.TrimSpace(customErrMsg + "\n" + translatedErrMsg)

					if errMsg != "" {
						rt = result.SimpleFail[string](errMsg)
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

func getCustomErrMsg(req any, errNss map[string]validator.FieldError) (string, []validator.FieldError) {
	reqType := reflect.TypeOf(req)
	if reqType.Kind() != reflect.Ptr || reqType.Elem().Kind() != reflect.Struct {
		return "", getFieldErrors(errNss)
	}

	errMsgs := findErrMsgs(reqType.Elem(), reqType.Elem().Name(), errNss, "")
	errMsg := strings.Join(errMsgs, "\n")
	otherErrs := getFieldErrors(errNss)
	return errMsg, otherErrs
}

func findErrMsgs(tType reflect.Type, tName string, errNss map[string]validator.FieldError, t string) []string {
	var ret []string

	var sType reflect.Type
	sTypeKind := tType.Kind()
	if sTypeKind == reflect.Ptr {
		sType = tType.Elem()
	} else {
		sType = tType
	}

	t = t + tName + "."

	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		fieldName := field.Name
		ns := t + fieldName
		_, ok := errNss[ns]
		if ok {
			tag := field.Tag.Get("errMsg")
			if tag != "" {
				ret = append(ret, tag)
				delete(errNss, ns)
			}
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			subret := findErrMsgs(fieldType, fieldName, errNss, t)
			if len(subret) > 0 {
				ret = append(ret, subret...)
			}
		}
	}
	return ret
}

func getFieldErrors(errNss map[string]validator.FieldError) []validator.FieldError {
	var errs []validator.FieldError
	if len(errNss) > 0 {
		for _, v := range errNss {
			errs = append(errs, v)
		}
	}
	return errs
}

func getTranslateErrMsg(errs []validator.FieldError, lng string) string {
	if len(errs) == 0 {
		return ""
	}

	return ve.TranslateValidateError(validator.ValidationErrors(errs), lng)
}
