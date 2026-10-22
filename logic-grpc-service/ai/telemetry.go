package ai

import (
	"context"
	"errors"

	"logic-grpc-service/telemetry"

	"github.com/cloudwego/eino-ext/components/model/openai"
	provider "github.com/meguminnnnnnnnn/go-openai"
	"go.opentelemetry.io/otel"
)

var (
	aiMeter         = otel.Meter("recruitment")
	requestCalls, _ = aiMeter.Int64Histogram("recruitment.ai.request.model.attempts")
	agentTools, _   = aiMeter.Int64Histogram("recruitment.agent.tool.calls")
)

func modelOutcome(err error) string {
	if errors.Is(err, ErrModelCircuitOpen) {
		return "circuit_open"
	}
	if errors.Is(err, ErrTokenBudgetExceeded) {
		return "budget_exceeded"
	}
	code := 0
	var api *openai.APIError
	var req *provider.RequestError
	if errors.As(err, &api) {
		code = api.HTTPStatusCode
	} else if errors.As(err, &req) {
		code = req.HTTPStatusCode
	}
	switch {
	case code == 429:
		return "rate_limited"
	case code == 408 || code == 504:
		return "timeout"
	case code >= 500:
		return "provider_error"
	case code >= 400:
		return "request_error"
	default:
		return telemetry.Outcome(err)
	}
}

func recordModelAttempts(ctx context.Context, m *budgetModel) {
	m.budget.mu.Lock()
	n := m.budget.calls
	m.budget.mu.Unlock()
	requestCalls.Record(ctx, int64(n))
}
