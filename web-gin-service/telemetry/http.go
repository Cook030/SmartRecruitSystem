package telemetry

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	meter         = otel.Meter("recruitment")
	operations, _ = meter.Int64Counter("recruitment.operations")
	duration, _   = meter.Float64Histogram("recruitment.operation.duration", metric.WithUnit("s"))
	inflight, _   = meter.Int64UpDownCounter("recruitment.operations.inflight")
)

// otelgin owns the server span; this middleware records stable route labels.
func HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		method := c.Request.Method
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		default:
			method = "OTHER"
		}
		attrs := []attribute.KeyValue{attribute.String("operation", "http.server"), attribute.String("route", route), attribute.String("method", method)}
		ctx := c.Request.Context()
		inflight.Add(ctx, 1, metric.WithAttributes(attrs...))
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			c.Header("X-Trace-ID", sc.TraceID().String())
		}
		defer func() {
			inflight.Add(ctx, -1, metric.WithAttributes(attrs...))
			code := c.Writer.Status()
			outcome := "success"
			if code >= 400 {
				outcome = "client_error"
			}
			if code >= 500 {
				outcome = "error"
			}
			attrs = append(attrs, attribute.String("status", strconv.Itoa(code)), attribute.String("outcome", outcome))
			operations.Add(ctx, 1, metric.WithAttributes(attrs...))
			duration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(attrs...))
			slog.InfoContext(ctx, "http request finished", "method", method, "route", route, "status", code, "duration_ms", time.Since(started).Milliseconds())
		}()
		c.Next()
	}
}
