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
	"mime/multipart"
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

	if len(body) > 0 {
		if c.ContentType() == "multipart/form-data" {
			err := parseMultipartForm(c, body, mp)
			if err != nil {
				dglogger.Errorf(ctx, "form data parse error: %v", err)
			}
		} else {
			err := json.Unmarshal(body, &mp)
			if err != nil {
				dglogger.Errorf(ctx, "parse request body error: %v", err)
			}
		}
	}

	if len(c.Request.URL.Query()) > 0 {
		for k := range c.Request.URL.Query() {
			mp[k] = c.Query(k)
		}
	}

	return mp
}

func parseMultipartForm(c *gin.Context, body []byte, mp map[string]any) error {
	reader := multipart.NewReader(bytes.NewReader(body), c.ContentType())
	form, err := reader.ReadForm(32 << 20) // 32 MB max memory
	if err != nil {
		return err
	}
	for key, values := range form.Value {
		if len(values) > 0 {
			mp[key] = values[0]
		}
	}
	return nil
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
	if traceId != "" {
		return traceId
	}

	traceId = c.Query(constants.TraceId)
	if traceId != "" {
		return traceId
	}

	return uuid.NewString()
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
