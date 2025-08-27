package wrapper

import (
	"encoding/json"
	"go/types"
	"net/http"
	"strings"
	"time"

	dgcoll "github.com/darwinOrg/go-common/collection"
	"github.com/darwinOrg/go-common/constants"
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/result"
	dgsys "github.com/darwinOrg/go-common/sys"
	dghttp "github.com/darwinOrg/go-httpclient"
	dglogger "github.com/darwinOrg/go-logger"
	dgotel "github.com/darwinOrg/go-otel"
	ve "github.com/darwinOrg/go-validator-ext"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"go.opentelemetry.io/otel/trace"
)

type LogLevel int

const (
	LOG_LEVEL_NONE    LogLevel = -1
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

var RequestApis []*RequestApi

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
	Remark           string
	RelativePath     string
	PreHandlersChain gin.HandlersChain
	NonLogin         bool
	AllowRoles       []string
	AllowProducts    []int
	NeedPermissions  []string
	BizHandler       HandlerFunc[T, V]
	LogLevel         LogLevel
	NotLogSQL        bool
	EnableTracer     bool
}

type EmptyRequest struct{}

var emptyRequest = new(EmptyRequest)

type HandlerFunc[T any, V any] func(gc *gin.Context, dc *dgctx.DgContext, requestObj *T) V

func Get[T any, V any](rh *RequestHolder[T, V]) {
	rh.GET(rh.RelativePath, BuildHandlersChain(rh)...)
	AppendRequestApi(rh, http.MethodGet)
}

func Post[T any, V any](rh *RequestHolder[T, V]) {
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
		if rh.LogLevel == 0 {
			rh.LogLevel = DEFAULT_LOG_LEVEL
		}

		ctx := utils.GetDgContext(c)
		ctx.NotLogSQL = rh.NotLogSQL
		ctx.EnableTracer = rh.EnableTracer && dgotel.Tracer != nil
		if ctx.EnableTracer {
			if span := trace.SpanFromContext(c.Request.Context()); span.SpanContext().IsValid() {
				attrs := dghttp.ExtractOtelAttributesFromRequest(c.Request)
				if len(attrs) > 0 {
					span.SetAttributes(attrs...)
				}

				defer func() {
					span.SetAttributes(semconv.HTTPResponseContentLength(c.Writer.Size()))
				}()
			}
		}

		var rt any
		req := new(T)
		if err := c.ShouldBind(req); err != nil {
			dglogger.Errorf(ctx, "bind request object error: %v", err)
			errMsg := ve.TranslateValidateError(err, ctx.Lang)
			if errMsg != "" {
				rt = result.SimpleFailByError(dgerr.SimpleDgError(errMsg))
			} else {
				rt = result.SimpleFailByError(err)
			}
		} else {
			rt = rh.BizHandler(c, ctx, req)
		}

		if len(returnResultPostProcessors) > 0 {
			for _, returnResultPostProcessor := range returnResultPostProcessors {
				returnResultPostProcessor(ctx, rt)
			}
		}

		printBizHandlerLog(c, ctx, req, rt, start, rh.LogLevel)

		if !c.Writer.Written() {
			c.JSON(http.StatusOK, rt)
		}

		c.Next()
	}
}

func printBizHandlerLog[T any](c *gin.Context, ctx *dgctx.DgContext, rp *T, rt any, start time.Time, ll LogLevel) {
	if ll == LOG_LEVEL_NONE {
		return
	}

	ctxJson, _ := json.Marshal(ctx)
	latency := time.Now().Sub(start)

	if ll == LOG_LEVEL_ALL {
		rpBytes, _ := dglogger.Json(rp)
		rtBytes, _ := dglogger.Json(rt)
		dglogger.Infof(ctx, "path: %s, context: %s, params: %s, result: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, rtBytes, latency)
	} else if ll == LOG_LEVEL_PARAM {
		rpBytes, _ := dglogger.Json(rp)
		dglogger.Infof(ctx, "path: %s, context: %s, params: %s, cost: %13v", c.Request.URL.Path, ctxJson, rpBytes, latency)
	} else if ll == LOG_LEVEL_RETURN {
		rtBytes, _ := dglogger.Json(rt)
		dglogger.Infof(ctx, "path: %s, context: %s, result: %s, cost: %13v", c.Request.URL.Path, ctxJson, rtBytes, latency)
	}
}

func AppendRequestApi[T any, V any](rh *RequestHolder[T, V], method string) {
	RequestApis = append(RequestApis, &RequestApi{
		Method:         method,
		BasePath:       rh.BasePath(),
		RelativePath:   rh.RelativePath,
		Remark:         rh.Remark,
		RequestObject:  new(T),
		ResponseObject: new(V),
	})
}

func GetRequestApis() []*RequestApi {
	return RequestApis
}
