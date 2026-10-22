package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestModelMetricsCountUsageAndBudgetRejection(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)
	defer mp.Shutdown(ctx)
	fake := &meteredFake{usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}}
	model := &circuitModel{ToolCallingChatModel: fake, circuit: &modelCircuit{}, name: "test-model"}
	m := withTokenBudget(model, 6000)
	if _, err := m.Generate(ctx, []*schema.Message{schema.UserMessage("private-prompt")}); err != nil {
		t.Fatal(err)
	}
	blocked := withTokenBudget(model, 1)
	if _, err := blocked.Generate(ctx, nil); !errors.Is(err, ErrTokenBudgetExceeded) {
		t.Fatal("budget guard bypassed")
	}
	if fake.calls != 1 {
		t.Fatal("budget rejection called provider")
	}
	var snapshot metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &snapshot); err != nil {
		t.Fatal(err)
	}
	usage := map[string]int64{}
	var budgetEvents int64
	for _, scope := range snapshot.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name == "recruitment.ai.tokens" {
				for _, point := range metric.Data.(metricdata.Sum[int64]).DataPoints {
					kind, _ := point.Attributes.Value("kind")
					usage[kind.AsString()] += point.Value
				}
			}
			if metric.Name == "recruitment.events" {
				for _, point := range metric.Data.(metricdata.Sum[int64]).DataPoints {
					name, _ := point.Attributes.Value("event")
					if name.AsString() == "ai.token_budget_exceeded" {
						budgetEvents += point.Value
					}
				}
			}
		}
	}
	if usage["input"] != 100 || usage["output"] != 50 || usage["total"] != 150 || budgetEvents != 1 {
		t.Fatalf("incorrect accounting: usage=%v budget=%d", usage, budgetEvents)
	}
	for _, tc := range []struct {
		err  error
		want string
	}{
		{&openai.APIError{HTTPStatusCode: 429}, "rate_limited"},
		{&openai.APIError{HTTPStatusCode: 503}, "provider_error"},
		{context.DeadlineExceeded, "timeout"},
		{ErrModelCircuitOpen, "circuit_open"},
	} {
		if got := modelOutcome(tc.err); got != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, got)
		}
	}
}
