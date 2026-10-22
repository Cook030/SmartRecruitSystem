package telemetry

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	meter          = otel.Meter("recruitment")
	operations, _  = meter.Int64Counter("recruitment.operations")
	duration, _    = meter.Float64Histogram("recruitment.operation.duration", metric.WithUnit("s"))
	inflight, _    = meter.Int64UpDownCounter("recruitment.operations.inflight")
	events, _      = meter.Int64Counter("recruitment.events")
	tokens, _      = meter.Int64Counter("recruitment.ai.tokens")
	taskLatency, _ = meter.Float64Histogram("recruitment.task.latency", metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120, 300, 600, 1800))
)

// Rare-event counters need a zero sample before their first increment so that
// Prometheus increase() can detect the first failure after a successful scrape.
func InitBusinessCounters(ctx context.Context, model string) {
	for _, name := range []string{"task.enqueue_failed", "task.state_update_failed", "ai.token_budget_exceeded", "ai.circuit_rejected", "ai.step_budget_exceeded", "ai.invalid_score", "context.degraded", "chat.replayed"} {
		events.Add(ctx, 0, metric.WithAttributes(attribute.String("event", name)))
	}
	events.Add(ctx, 0, metric.WithAttributes(attribute.String("event", "ai.usage_missing"), attribute.String("model", model)))
	for _, kind := range []string{"resume_parse", "ai_match", "stats_refresh", "notification", "other"} {
		for _, state := range []string{"success", "failed", "retrying"} {
			events.Add(ctx, 0, metric.WithAttributes(attribute.String("event", "task."+state), attribute.String("task_type", kind)))
		}
	}
	for _, kind := range []string{"resume_parse", "ai_match"} {
		events.Add(ctx, 0, metric.WithAttributes(attribute.String("event", "ai.invalid_json"), attribute.String("task_type", kind)))
	}
}

// Start measures one attempt. outcome must be a controlled code, never an error string.
func Start(ctx context.Context, operation string, attrs ...attribute.KeyValue) (context.Context, func(string)) {
	attrs = append([]attribute.KeyValue{attribute.String("operation", operation)}, attrs...)
	ctx, span := otel.Tracer("recruitment").Start(ctx, operation, trace.WithAttributes(attrs...))
	started := time.Now()
	inflight.Add(ctx, 1, metric.WithAttributes(attrs...))
	return ctx, func(outcome string) {
		inflight.Add(ctx, -1, metric.WithAttributes(attrs...))
		final := append(append([]attribute.KeyValue{}, attrs...), attribute.String("outcome", outcome))
		operations.Add(ctx, 1, metric.WithAttributes(final...))
		duration.Record(ctx, time.Since(started).Seconds(), metric.WithAttributes(final...))
		span.SetAttributes(attribute.String("outcome", outcome))
		if outcome != "success" && outcome != "miss" && outcome != "not_found" {
			span.SetStatus(codes.Error, outcome)
		}
		span.End()
	}
}

func Outcome(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	default:
		return "error"
	}
}

func Event(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	events.Add(ctx, 1, metric.WithAttributes(append(attrs, attribute.String("event", name))...))
}

func Tokens(ctx context.Context, model, kind string, count int) {
	if count > 0 {
		tokens.Add(ctx, int64(count), metric.WithAttributes(attribute.String("model", model), attribute.String("kind", kind)))
	}
}

func TaskLatency(ctx context.Context, taskType, phase string, elapsed time.Duration) {
	taskLatency.Record(ctx, max(0, elapsed.Seconds()), metric.WithAttributes(attribute.String("task_type", taskType), attribute.String("phase", phase)))
}

// Task types can also come from RPC input; keep metric cardinality bounded.
func TaskType(value string) string {
	switch value {
	case "resume_parse", "ai_match", "stats_refresh", "notification":
		return value
	default:
		return "other"
	}
}
