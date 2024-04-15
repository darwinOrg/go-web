package utils

import (
	"bytes"
	"encoding/json"
	dgcoll "github.com/darwinOrg/go-common/collection"
	"github.com/darwinOrg/go-common/constants"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const DgContextKey = "DgContext"

func GetLang(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	lng := GetHeader(c, constants.Lang)
	if lng != "" {
		return lng
	}

	lng = GetHeader(c, "Accept-Language")
	if lng != "" {
		return lng
	}

	return c.Query(constants.Lang)
}

func GetAllRequestParams(c *gin.Context, ctx *dgctx.DgContext) map[string]any {
	var body []byte

	if !isGetOrHead(c) {
		if cb, ok := c.Get(gin.BodyBytesKey); ok {
			if cbb, ok := cb.([]byte); ok {
				body = cbb
			}
		}

		if len(body) == 0 {
			body, _ = io.ReadAll(c.Request.Body)
			if len(body) > 0 {
				c.Set(gin.BodyBytesKey, body)
				c.Request.Body = io.NopCloser(bytes.NewReader(body))
			}
		}
	}

	mp := map[string]any{}

	if body != nil {
		err := json.Unmarshal(body, &mp)
		if err != nil {
			dglogger.Infof(ctx, "parse request body error: %v", err)
		}
	}

	if len(c.Request.URL.Query()) > 0 {
		for k := range c.Request.URL.Query() {
			mp[k] = c.Query(k)
		}
	}

	return mp
}

func isGetOrHead(c *gin.Context) bool {
	return strings.EqualFold(c.Request.Method, "GET") ||
		strings.EqualFold(c.Request.Method, "HEAD")
}

func GetDgContext(c *gin.Context) *dgctx.DgContext {
	ctx, ok := c.Get(DgContextKey)
	if !ok {
		ctx = BuildDgContext(c)
		c.Set(DgContextKey, ctx)
	}
	return ctx.(*dgctx.DgContext)
}

func BuildDgContext(c *gin.Context) *dgctx.DgContext {
	return &dgctx.DgContext{
		TraceId:       GetTraceId(c),
		UserId:        GetUserId(c),
		OpId:          getInt64Value(c, constants.OpId),
		RunAs:         getInt64Value(c, constants.RunAs),
		Roles:         GetHeader(c, constants.Roles),
		BizTypes:      getIntValue(c, constants.BizTypes),
		GroupId:       getInt64Value(c, constants.GroupId),
		Platform:      GetPlatform(c),
		UserAgent:     GetHeader(c, constants.UserAgent),
		Lang:          GetLang(c),
		Token:         GetToken(c),
		ShareToken:    GetShareToken(c),
		RemoteIp:      GetHeader(c, constants.RemoteIp),
		CompanyId:     getInt64Value(c, constants.CompanyId),
		Product:       GetProduct(c),
		Products:      GetProducts(c),
		DepartmentIds: GetDepartmentIds(c),
	}
}

func GetTraceId(c *gin.Context) string {
	traceId := GetHeader(c, constants.TraceId)
	if traceId == "" {
		traceId = uuid.NewString()
	}
	return traceId
}

func GetUserId(c *gin.Context) int64 {
	return getInt64Value(c, constants.UID)
}

func GetToken(c *gin.Context) string {
	token := GetHeader(c, constants.Token)
	if len(token) == 0 {
		token = c.Query(constants.Token)
	}
	return token
}

func GetPlatform(c *gin.Context) string {
	platform := GetHeader(c, constants.Platform)
	if len(platform) == 0 {
		platform = c.Query(constants.Platform)
	}
	return platform
}

func GetShareToken(c *gin.Context) string {
	token := GetHeader(c, constants.ShareToken)
	if len(token) == 0 {
		token = c.Query(constants.ShareToken)
	}
	return token
}

func GetProduct(c *gin.Context) int {
	product := GetHeader(c, constants.Product)
	if len(product) == 0 {
		product = c.Query(constants.Product)
	}
	if len(product) == 0 {
		return 0
	}
	val, _ := strconv.Atoi(product)
	return val
}

func GetHeader(c *gin.Context, key string) string {
	header := c.GetHeader(key)
	if header != "" {
		return header
	}

	headers := c.Request.Header[key]
	if len(headers) > 0 {
		return headers[0]
	}

	return ""
}

func GetProducts(c *gin.Context) []int {
	products := GetHeader(c, constants.Products)
	if len(products) > 0 {
		return dgcoll.SplitToIntsByComma[int](products)
	}
	return []int{}
}

func GetDepartmentIds(c *gin.Context) []int64 {
	departmentIds := GetHeader(c, constants.DepartmentIds)
	if len(departmentIds) > 0 {
		return dgcoll.SplitToIntsByComma[int64](departmentIds)
	}
	return []int64{}
}

func getInt64Value(c *gin.Context, header string) int64 {
	h := GetHeader(c, header)
	if h == "" {
		h = "0"
	}
	val, _ := strconv.ParseInt(h, 10, 64)
	return val
}

func getIntValue(c *gin.Context, header string) int {
	h := GetHeader(c, header)
	if h == "" {
		h = "0"
	}
	val, _ := strconv.Atoi(h)
	return val
}

func AppendUidForUrlValue(ctx *dgctx.DgContext, obj any) {
	appendUidForUrlValue(ctx, reflect.TypeOf(obj), reflect.ValueOf(obj))
}

func appendUidForUrlValue(ctx *dgctx.DgContext, tpe reflect.Type, elem reflect.Value) {
	for tpe.Kind() == reflect.Pointer {
		tpe = tpe.Elem()
		elem = elem.Elem()
	}

	if tpe.Kind() == reflect.Struct {
		cnt := tpe.NumField()

		for i := 0; i < cnt; i++ {
			field := tpe.Field(i)
			fieldType := field.Type
			fieldValue := elem.Field(i)
			for fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
				fieldValue = fieldValue.Elem()
			}

			if field.Tag.Get("appendUid") == "true" {
				if fieldType.Kind() == reflect.String && fieldValue.IsValid() {
					fieldString := fieldValue.String()
					if fieldString != "" {
						if strings.Contains(fieldString, constants.UID+"=") {
							continue
						}
						if strings.Contains(fieldString, "?") {
							fieldString += "&"
						} else {
							fieldString += "?"
						}
						fieldString += constants.UID + "=" + strconv.FormatInt(ctx.UserId, 10)
						fieldValue.SetString(fieldString)
					}
				}
			} else if fieldType.Kind() == reflect.Struct {
				appendUidForUrlValue(ctx, fieldType, fieldValue)
			}
		}
	}
}
