package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	provider "github.com/meguminnnnnnnnn/go-openai"
)

func TestModelCircuitRecovery(t *testing.T) {
	b := &modelCircuit{}
	now := time.Now()
	for i := 0; i < modelFailureThreshold; i++ {
		generation, err := b.allow(now)
		if err != nil {
			t.Fatal(err)
		}
		b.finish(generation, context.DeadlineExceeded, now)
	}
	if _, err := b.allow(now); !errors.Is(err, ErrModelCircuitOpen) {
		t.Fatal("circuit did not open")
	}
	// A late successful request from before opening cannot reset the circuit.
	b.finish(0, nil, now)
	if _, err := b.allow(now); !errors.Is(err, ErrModelCircuitOpen) {
		t.Fatal("stale success closed circuit")
	}
	now = now.Add(modelCircuitCooldown)
	var admitted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Go(func() {
			if _, err := b.allow(now); err == nil {
				admitted.Add(1)
			}
		})
	}
	wg.Wait()
	if admitted.Load() != 1 {
		t.Fatalf("admitted %d probes", admitted.Load())
	}
	b.finish(b.generation, context.DeadlineExceeded, now)
	if _, err := b.allow(now); !errors.Is(err, ErrModelCircuitOpen) {
		t.Fatal("failed probe did not reopen")
	}
	now = now.Add(modelCircuitCooldown)
	generation, err := b.allow(now)
	if err != nil {
		t.Fatal(err)
	}
	b.finish(generation, nil, now)
	if _, err := b.allow(now); err != nil {
		t.Fatal("successful probe did not restore traffic")
	}
}

func TestModelCircuitProviderErrors(t *testing.T) {
	if !modelProviderFailure(&provider.RequestError{HTTPStatusCode: 502}) {
		t.Fatal("proxy failure was ignored")
	}
	for _, code := range []int{400, 401, 403, 408, 429, 500, 503} {
		err := fmt.Errorf("wrapped: %w", &openai.APIError{HTTPStatusCode: code})
		want := code == 408 || code == 429 || code >= 500
		if modelProviderFailure(err) != want {
			t.Fatalf("wrong classification for %d", code)
		}
	}
	if modelProviderFailure(context.Canceled) || modelProviderFailure(ErrTokenBudgetExceeded) {
		t.Fatal("local errors counted as provider failures")
	}
	b := &modelCircuit{}
	now := time.Now()
	for i := 0; i < 4; i++ {
		b.finish(0, context.DeadlineExceeded, now)
	}
	b.finish(0, nil, now)
	b.finish(0, context.DeadlineExceeded, now)
	if _, err := b.allow(now); err != nil {
		t.Fatal("success did not reset consecutive failures")
	}
}

func TestCircuitModelBlocksProviderAndKeepsTools(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"error":{"message":"unavailable","type":"server_error"}}`)
	}))
	defer server.Close()
	inner, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{BaseURL: server.URL + "/v1", APIKey: "test", Model: "test"})
	if err != nil {
		t.Fatal(err)
	}
	b := &modelCircuit{}
	m := &circuitModel{ToolCallingChatModel: inner, circuit: b}
	bound, err := m.WithTools([]*schema.ToolInfo{{Name: "test_tool", Desc: "test", ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{})}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < modelFailureThreshold; i++ {
		if _, err := bound.Generate(context.Background(), []*schema.Message{schema.UserMessage("test")}); err == nil {
			t.Fatal("expected provider error")
		}
	}
	if _, err := m.Generate(context.Background(), nil); !errors.Is(err, ErrModelCircuitOpen) {
		t.Fatal("bound model lost shared circuit")
	}
	if calls.Load() != modelFailureThreshold {
		t.Fatalf("unexpected provider calls: %d", calls.Load())
	}
}
