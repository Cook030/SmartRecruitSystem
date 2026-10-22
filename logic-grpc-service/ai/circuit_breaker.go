package ai

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"logic-grpc-service/telemetry"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	provider "github.com/meguminnnnnnnnn/go-openai"
	"go.opentelemetry.io/otel/attribute"
)

var ErrModelCircuitOpen = errors.New("AI service temporarily unavailable; please retry later")

const modelFailureThreshold = 5
const modelCircuitCooldown = 30 * time.Second

// One configured provider: all requests in this process share its circuit.
// Replicas independently protect their own outgoing calls.
var providerCircuit = &modelCircuit{}

type modelCircuit struct {
	mu         sync.Mutex
	failures   int
	until      time.Time
	probing    bool
	generation uint64
}

func (b *modelCircuit) allow(now time.Time) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.until.IsZero() {
		if now.Before(b.until) || b.probing {
			return 0, ErrModelCircuitOpen
		}
		b.probing = true
	}
	return b.generation, nil
}

func (b *modelCircuit) finish(generation uint64, err error, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// An old in-flight call cannot change a newer circuit/probe.
	if generation != b.generation {
		return
	}
	if b.probing {
		b.probing = false
		b.generation++
		if err != nil {
			// Only a successful probe restores traffic, even if its caller cancels.
			b.until = now.Add(modelCircuitCooldown)
			return
		}
		b.until = time.Time{}
		b.failures = 0
		slog.Info("AI circuit closed")
		return
	}
	if !modelProviderFailure(err) {
		if err == nil {
			b.failures = 0
		}
		return
	}
	b.failures++
	if b.failures >= modelFailureThreshold {
		b.until = now.Add(modelCircuitCooldown)
		b.generation++
		slog.Warn("AI circuit opened", "cooldown", modelCircuitCooldown)
	}
}

func modelProviderFailure(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var api *openai.APIError
	if errors.As(err, &api) {
		return api.HTTPStatusCode == 429 || api.HTTPStatusCode == 408 || api.HTTPStatusCode >= 500
	}
	// Proxies may return HTML/plain text instead of an OpenAI error body.
	var request *provider.RequestError
	if errors.As(err, &request) && request.HTTPStatusCode != 0 {
		return request.HTTPStatusCode == 429 || request.HTTPStatusCode == 408 || request.HTTPStatusCode >= 500
	}
	var network net.Error
	return errors.As(err, &network)
}

type circuitModel struct {
	model.ToolCallingChatModel
	circuit *modelCircuit
	name    string
}

func withModelCircuit(inner model.ToolCallingChatModel, names ...string) *circuitModel {
	name := "unknown"
	if len(names) > 0 {
		name = names[0]
	}
	return &circuitModel{ToolCallingChatModel: inner, circuit: providerCircuit, name: name}
}

func (m *circuitModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.ToolCallingChatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &circuitModel{ToolCallingChatModel: inner, circuit: m.circuit, name: m.name}, nil
}

func (m *circuitModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	generation, err := m.circuit.allow(time.Now())
	if err != nil {
		telemetry.Event(ctx, "ai.circuit_rejected")
		return nil, err
	}
	// Background analysis otherwise has no request deadline.
	ctx, cancel := context.WithTimeout(ctx, ChatTimeout)
	defer cancel()
	ctx, end := telemetry.Start(ctx, "ai.model", attribute.String("model", m.name))
	response, err := m.ToolCallingChatModel.Generate(ctx, messages, opts...)
	end(modelOutcome(err))
	if response != nil && response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		usage := response.ResponseMeta.Usage
		telemetry.Tokens(ctx, m.name, "input", usage.PromptTokens)
		telemetry.Tokens(ctx, m.name, "output", usage.CompletionTokens)
		telemetry.Tokens(ctx, m.name, "total", max(usage.TotalTokens, usage.PromptTokens+usage.CompletionTokens))
	} else {
		telemetry.Event(ctx, "ai.usage_missing", attribute.String("model", m.name))
	}
	m.circuit.finish(generation, err, time.Now())
	return response, err
}

// Current callers use Generate. Reject streaming instead of bypassing protection.
func (m *circuitModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("model circuit does not support streaming")
}
