package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	promexport "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestTaskTraceDatabaseAndMetricExport(t *testing.T) {
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	exporter, err := promexport.New(promexport.WithRegisterer(registry))
	if err != nil {
		t.Fatal(err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(mp)
	defer mp.Shutdown(ctx)
	InitBusinessCounters(ctx, "test-model")
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(ctx)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	parent, end := Start(ctx, "task.create")
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(parent, carrier)
	end("success")
	task, finish := Start(otel.GetTextMapPropagator().Extract(ctx, carrier), "task.execute", attribute.String("task_type", "resume_parse"))
	// A dry-run query executes the real GORM callbacks without a database.
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "unused:unused@tcp(127.0.0.1:1)/unused", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Use(DatabasePlugin{}); err != nil {
		t.Fatal(err)
	}
	var result []struct{ ID int }
	query := db.WithContext(task).Table("private_table").Where("secret = ?", "private-value").Find(&result)
	if query.Error != nil {
		t.Fatal(query.Error)
	}
	Tokens(task, "test-model", "input", 20)
	Event(task, "task.retrying", attribute.String("task_type", "resume_parse"))
	TaskLatency(task, "resume_parse", "end_to_end", time.Second)
	var logs bytes.Buffer
	logger := slog.New(traceHandler{slog.NewJSONHandler(&logs, nil)})
	logger.InfoContext(task, "task completed")
	finish("error")
	spans := recorder.Ended()
	if len(spans) != 3 {
		t.Fatalf("expected create, execute, query; got %d", len(spans))
	}
	for _, span := range spans {
		if span.SpanContext().TraceID() != spans[0].SpanContext().TraceID() {
			t.Fatal("async trace disconnected")
		}
		for _, a := range span.Attributes() {
			if strings.Contains(a.Value.AsString(), "private") {
				t.Fatal("SQL data leaked into span")
			}
		}
	}
	if !strings.Contains(logs.String(), spans[0].SpanContext().TraceID().String()) {
		t.Fatal("log trace ID missing")
	}
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	for _, want := range []string{"recruitment_operations_total", "recruitment_operation_duration_seconds_bucket", "recruitment_events_total", "recruitment_ai_tokens_total", "recruitment_task_latency_seconds_bucket", `operation="db.query"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing metric %s: %s", want, body)
		}
	}
	if strings.Contains(body, "private") {
		t.Fatal("SQL data leaked into metrics")
	}
}
