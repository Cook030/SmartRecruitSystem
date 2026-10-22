package telemetry

import (
	"context"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	promexport "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

// No database, provider, collector or TCP socket is required.
func TestHTTPToGRPCTraceAndMetrics(t *testing.T) {
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	exporter, err := promexport.New(promexport.WithRegisterer(registry))
	if err != nil {
		t.Fatal(err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(mp)
	defer mp.Shutdown(ctx)
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer tp.Shutdown(ctx)

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	go server.Serve(listener)
	defer server.Stop()
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(otelgin.Middleware("test-web"), HTTPMetrics(), gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) { c.AbortWithStatus(500) }))
	router.GET("/tasks/:id", func(c *gin.Context) {
		_, err := grpc_health_v1.NewHealthClient(conn).Check(c.Request.Context(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			t.Error(err)
			c.Status(500)
			return
		}
		c.Status(200)
	})
	router.GET("/panic", func(*gin.Context) { panic("test recovery") })
	for _, path := range []string{"/tasks/private-123", "/tasks/private-456", "/missing-private", "/panic"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Header().Get("X-Trace-ID") == "" {
			t.Fatal("trace correlation header missing")
		}
	}
	spans := recorder.Ended()
	matched := 0
	// Each successful request has an HTTP, gRPC client and gRPC server span.
	for _, span := range spans {
		if span.Name() == "GET /tasks/:id" {
			matched++
			n := 0
			for _, candidate := range spans {
				if candidate.SpanContext().TraceID() == span.SpanContext().TraceID() {
					n++
				}
			}
			if n != 3 {
				t.Fatalf("HTTP/gRPC trace disconnected: got %d spans", n)
			}
		}
	}
	if matched != 2 {
		t.Fatalf("expected two HTTP request spans, got %d", matched)
	}
	if len(spans) != 8 {
		t.Fatalf("expected 8 spans, got %d", len(spans))
	}
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	for _, want := range []string{"recruitment_operations_total", "recruitment_operation_duration_seconds_bucket", `route="/tasks/:id"`, `route="unmatched"`, `outcome="error"`, `status="500"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in metrics: %s", want, body)
		}
	}
	if strings.Contains(body, "private-") {
		t.Fatal("raw path leaked into metric labels")
	}
}
