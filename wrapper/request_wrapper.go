package wrapper

import (
	"encoding/json"
	"errors"
	dgcoll "github.com/darwinOrg/go-common/collection"
	"github.com/darwinOrg/go-common/constants"
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
	LOG_LEVEL_NONE    LogLevel = 0
	LOG_LEVEL_PARAM   LogLevel = 1
	LOG_LEVEL_RETURN  LogLevel = 2
	LOG_LEVEL_ALL     LogLevel = 3
	DEFAULT_LOG_LEVEL          = LOG_LEVEL_ALL
)

var myProfile = dgsys.GetProfile()

var (
	EnableRolesCheck    = true
	EnableProductsCheck = true
)

type ReturnResultPostProcessor func(ctx *dgctx.DgContext, rt any)

var returnResultPostProcessors []ReturnResultPostProcessor

func RegisterReturnResultPostProcessor(processor ReturnResultPostProcessor) {
	returnResultPostProcessors = append(returnResultPostProcessors, processor)
}

var (
	requestApis []*RequestApi
)

type RequestApi struct {
	Method         string
	BasePath       string
	RelativePath   string
	Remark         string
	RequestObject  any
	ResponseObject any
}

type RequestHolder[T any, V any] struct {
	*gin.RouterGroup
	RelativePath     string
	PreHandlersChain gin.HandlersChain
	NonLogin         bool
	AllowRoles       []string
	AllowProducts    []int
	NeedPermissions  []string
	BizHandler       HandlerFunc[T, V]
	mapRequestObj    bool
	LogLevel         LogLevel
	Remark           string
}

type MapRequest struct {
	MP map[string]any
}

type HandlerFunc[T any, V any] func(gc *gin.Context, dc *dgctx.DgContext, requestObj *T) V

func Get[T any, V any](rh *RequestHolder[T, V]) {
	rh.GET(rh.RelativePath, BuildHandlersChain(rh)...)
	AppendRequestApi(rh, http.MethodGet)
}

func Post[T any, V any](rh *RequestHolder[T, V]) {
	rh.POST(rh.RelativePath, BuildHandlersChain(rh)...)
	AppendRequestApi(rh, http.MethodPost)
}

func MapGet[V any](rh *RequestHolder[MapRequest, V]) {
	rh.mapRequestObj = true
	rh.GET(rh.RelativePath, BuildHandlersChain(rh)...)
	AppendRequestApi(rh, http.MethodGet)
}

func MapPost[V any](rh *RequestHolder[MapRequest, V]) {
	rh.mapRequestObj = true
	rh.POST(rh.RelativePath, BuildHandlersChain(rh)...)
	AppendRequestApi(rh, http.MethodPost)
}

func BuildHandlersChain[T any, V any](rh *RequestHolder[T, V]) gin.HandlersChain {
	handlersChain := []gin.HandlerFunc{LoginHandler(rh), CheckProductHandler(rh), CheckRolesHandler(rh), CheckProfileHandler(), BizHandler(rh)}

	if len(rh.PreHandlersChain) > 0 {
		return dgcoll.MergeToList(rh.PreHandlersChain, handlersChain)
	}

	return handlersChain
}

func LoginHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
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

func CheckProfileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		values := c.Request.Header[constants.Profile]
		if len(values) == 0 || len(values[0]) == 0 {
			c.Next()
			return
		}

		if values[0] != myProfile {
			ctx := utils.GetDgContext(c)
			dglogger.Warnf(ctx, "invalid profile, your profile is %s, current profile is %s", values[0], myProfile)
			c.AbortWithStatusJSON(http.StatusOK, result.SimpleFail[string]("your call incorrect profile"))
			return
		}

		c.Next()
	}
}

func CheckRolesHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !EnableRolesCheck || len(rh.AllowRoles) == 0 {
			c.Next()
			return
		}

		ctx := utils.GetDgContext(c)
		if ctx.Roles == "" {
			dglogger.Warnf(ctx, "has no roles")
			c.AbortWithStatusJSON(http.StatusOK, result.FailByError[types.Nil](dgerr.NO_PERMISSION))
			return
		}

		roles := strings.Split(ctx.Roles, ",")
		dgcoll.Intersection(roles, rh.AllowRoles)
		if !dgcoll.ContainsAny(roles, rh.AllowRoles) {
			dglogger.Warnf(ctx, "has no allowed roles")
			c.AbortWithStatusJSON(http.StatusOK, result.FailByError[types.Nil](dgerr.NO_PERMISSION))
			return
		}

		c.Next()
	}
}

func CheckProductHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !EnableProductsCheck || len(rh.AllowProducts) == 0 {
			c.Next()
			return
		}

		ctx := utils.GetDgContext(c)
		if len(ctx.Products) == 0 {
			dglogger.Warnf(ctx, "has no products")
			c.AbortWithStatusJSON(http.StatusOK, result.FailByError[*result.Void](dgerr.NO_PERMISSION))
			return
		}

		intersectionProducts := dgcoll.Intersection(ctx.Products, rh.AllowProducts)
		if len(intersectionProducts) == 0 {
			dglogger.Warnf(ctx, "has no allowed products")
			c.AbortWithStatusJSON(http.StatusOK, result.FailByError[*result.Void](dgerr.NO_PERMISSION))
			return
		}
		ctx.Product = intersectionProducts[0]

		c.Next()
	}
}

func BizHandler[T any, V any](rh *RequestHolder[T, V]) gin.HandlerFunc {
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
				dglogger.Errorf(ctx, "bind request object error: %v", err)

				var errs validator.ValidationErrors
				ok := errors.As(err, &errs)
				if ok {
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
					}
				}

				if rt == nil {
					rt = result.FailByError[*result.Void](dgerr.ARGUMENT_NOT_VALID)
				}

			} else {
				rt = rh.BizHandler(c, ctx, req)
			}
		}

		if len(returnResultPostProcessors) > 0 {
			for _, returnResultPostProcessor := range returnResultPostProcessors {
				returnResultPostProcessor(ctx, rt)
			}
		}

		printBizHandlerLog(c, ctx, rp, rt, start, rh.LogLevel)

		if !c.Writer.Written() {
			c.JSON(http.StatusOK, rt)
		}

		c.Next()
	}
}

func printBizHandlerLog(c *gin.Context, ctx *dgctx.DgContext, rp map[string]any, rt any, start time.Time, ll LogLevel) {
	if ll == LOG_LEVEL_NONE {
		return
	}

	ctxJson, _ := json.Marshal(ctx)
	latency := time.Now().Sub(start)

	if ll == LOG_LEVEL_ALL {
		rpBytes, _ := json.Marshal(rp)
		rtBytes, _ := json.Marshal(rt)
		dglogger.Infof(ctx, "path: %s, context: %s, params: %s, result: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, rtBytes, latency)
	} else if ll == LOG_LEVEL_PARAM {
		rpBytes, _ := json.Marshal(rp)
		dglogger.Infof(ctx, "path: %s, context: %s, params: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, latency)
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

func AppendRequestApi[T any, V any](rh *RequestHolder[T, V], method string) {
	if !dgsys.IsQa() && !dgsys.IsProd() {
		requestApis = append(requestApis, &RequestApi{
			Method:         method,
			BasePath:       rh.BasePath(),
			RelativePath:   rh.RelativePath,
			Remark:         rh.Remark,
			RequestObject:  new(T),
			ResponseObject: new(V),
		})
	}
}

func GetRequestApis() []*RequestApi {
	return requestApis
}
