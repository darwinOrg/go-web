package utils

import (
	"bytes"
	"encoding/json"
	"github.com/darwinOrg/go-common/constants"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"strconv"
	"strings"
)

const DgContextKey = "DgContext"

func GetLang(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	lng := c.GetHeader(constants.Lang)
	if lng != "" {
		return lng
	}

	lng = c.GetHeader("Accept-Language")
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
		TraceId:    GetTraceId(c),
		UserId:     GetUserId(c),
		OpId:       getInt64Value(c, constants.OpId),
		RunAs:      getInt64Value(c, constants.RunAs),
		Roles:      c.GetHeader(constants.Roles),
		BizTypes:   getIntValue(c, constants.BizTypes),
		GroupId:    getInt64Value(c, constants.GroupId),
		Platform:   c.GetHeader(constants.Platform),
		UserAgent:  c.GetHeader(constants.UserAgent),
		Lang:       GetLang(c),
		Token:      GetToken(c),
		ShareToken: GetShareToken(c),
		RemoteIp:   c.GetHeader(constants.RemoteIp),
		CompanyId:  getInt64Value(c, constants.CompanyId),
		Product:    getIntValue(c, constants.Product),
	}
}

func GetTraceId(c *gin.Context) string {
	traceId := c.GetHeader(constants.TraceId)
	if traceId == "" {
		traceId = uuid.NewString()
	}
	return traceId
}

func GetUserId(c *gin.Context) int64 {
	return getInt64Value(c, constants.UID)
}

func GetToken(c *gin.Context) string {
	token := c.Query(constants.Token)
	if len(token) == 0 {
		token = c.GetHeader(constants.Token)
	}
	return token
}

func GetShareToken(c *gin.Context) string {
	token := c.Query(constants.ShareToken)
	if len(token) == 0 {
		token = c.GetHeader(constants.ShareToken)
	}
	return token
}

func getInt64Value(c *gin.Context, header string) int64 {
	h := c.GetHeader(header)
	if h == "" {
		h = "0"
	}
	val, _ := strconv.ParseInt(h, 10, 64)
	return val
}

func getIntValue(c *gin.Context, header string) int {
	h := c.GetHeader(header)
	if h == "" {
		h = "0"
	}
	val, _ := strconv.Atoi(h)
	return val
}
