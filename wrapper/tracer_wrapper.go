package wrapper

import (
	"context"
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
	Tracer            trace.Tracer
	tracerServiceName string
	tracerMiddleware  gin.HandlerFunc
)

// InitTracer 初始化 OpenTelemetry 并配置 Jaeger 导出
func InitTracer(serviceName string, exporter *otlptrace.Exporter) (func(), error) {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)

	// 设置全局传播器（用于跨服务传递 TraceID 等）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Tracer = otel.Tracer("gin-server")
	tracerServiceName = serviceName
	tracerMiddleware = otelgin.Middleware(serviceName)

	// 返回一个关闭函数，用于优雅关闭 Tracer（比如 main结束时调用）
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("TracerProvider Shutdown Error: %v", err)
		}
	}, nil
}
