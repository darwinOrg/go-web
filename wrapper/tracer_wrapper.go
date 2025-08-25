package wrapper

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	Tracer           trace.Tracer
	tracerMiddleware gin.HandlerFunc
)

// InitTracer 初始化 OpenTelemetry 并配置导出
func InitTracer(serviceName string, exporter *otlptrace.Exporter) (func(), error) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Tracer = otel.Tracer(serviceName)
	tracerMiddleware = otelgin.Middleware(serviceName, otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
		return fmt.Sprintf("%s %s", c.Request.URL.Path, c.Request.Method)
	}))

	// 返回一个关闭函数，用于优雅关闭Tracer
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("TracerProvider Shutdown Error: %v", err)
		}
	}, nil
}
