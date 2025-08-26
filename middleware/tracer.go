package middleware

import (
	"github.com/darwinOrg/go-common/constants"
	dgotel "github.com/darwinOrg/go-otel"
	"github.com/darwinOrg/go-web/utils"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/trace"
)

func TraceId() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dgotel.Tracer == nil {
			c.Next()
			return
		}

		spanContext := trace.SpanContextFromContext(c.Request.Context())
		hasParentSpan := spanContext.IsValid() && spanContext.HasTraceID()

		if !hasParentSpan {
			traceId := utils.GetOrGenerateTraceId(c)
			traceIDFromHex, err := trace.TraceIDFromHex(traceId)
			if err == nil {
				// 构建新的 span context 配置
				config := trace.SpanContextConfig{
					TraceID: traceIDFromHex,
				}

				// 如果原 span context 有效，继承其他属性
				if spanContext.IsValid() {
					config.SpanID = spanContext.SpanID()
					config.TraceFlags = spanContext.TraceFlags()
					config.TraceState = spanContext.TraceState()
					config.Remote = spanContext.IsRemote()
				}

				// 创建新的 span context
				newSpanContext := trace.NewSpanContext(config)

				// 将新的 span context 注入到请求上下文中
				ctxWithTraceID := trace.ContextWithSpanContext(c.Request.Context(), newSpanContext)
				c.Request = c.Request.WithContext(ctxWithTraceID)

				// 设置响应头
				c.Header(constants.TraceId, traceIDFromHex.String())
			}
		} else {
			// 如果已有有效的 trace ID，直接使用
			c.Header(constants.TraceId, spanContext.TraceID().String())
		}

		c.Next()
	}
}

func Tracer(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName, otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
		return c.Request.URL.Path
	}))
}
