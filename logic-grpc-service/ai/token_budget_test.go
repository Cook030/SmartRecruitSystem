package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type meteredFake struct {
	model.ToolCallingChatModel
	calls int
	usage *schema.TokenUsage
	err   error
}

func (f *meteredFake) WithTools([]*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}
func (f *meteredFake) Generate(_ context.Context, _ []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	f.calls++
	if cap := model.GetCommonOptions(nil, opts...).MaxTokens; cap == nil || *cap > maxOutputTokens {
		panic("output cap bypassed")
	}
	return &schema.Message{ResponseMeta: &schema.ResponseMeta{Usage: f.usage}}, f.err
}

func TestTokenBudget(t *testing.T) {
	ctx := context.Background()
	messages := []*schema.Message{schema.UserMessage("hello")}
	t.Run("preflight", func(t *testing.T) {
		f := &meteredFake{}
		m := withTokenBudget(f, 1)
		if _, err := m.Generate(ctx, messages); !errors.Is(err, ErrTokenBudgetExceeded) || f.calls != 0 {
			t.Fatalf("preflight failed: calls=%d err=%v", f.calls, err)
		}
	})
	t.Run("settlement and shared tool model", func(t *testing.T) {
		f := &meteredFake{usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}}
		m := withTokenBudget(f, 6000)
		if _, err := m.Generate(ctx, messages, model.WithMaxTokens(99999)); err != nil {
			t.Fatal(err)
		}
		if m.budget.remaining != 5850 {
			t.Fatalf("remaining=%d", m.budget.remaining)
		}
		bound, err := m.WithTools([]*schema.ToolInfo{{Name: "search", Desc: "find candidates"}})
		if err != nil {
			t.Fatal(err)
		}
		f.usage = &schema.TokenUsage{TotalTokens: 6000}
		if _, err = bound.Generate(ctx, messages); !errors.Is(err, ErrTokenBudgetExceeded) {
			t.Fatalf("expected overage, got %v", err)
		}
		if _, err = m.Generate(ctx, messages); !errors.Is(err, ErrTokenBudgetExceeded) || f.calls != 2 {
			t.Fatalf("overage did not stop calls: calls=%d err=%v", f.calls, err)
		}
	})
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing usage", true: "provider error"}[failed], func(t *testing.T) {
			f := &meteredFake{}
			if failed {
				f.err = errors.New("provider failed")
			}
			m := withTokenBudget(f, 6000)
			m.Generate(ctx, messages)
			if _, err := m.Generate(ctx, messages); !errors.Is(err, ErrTokenBudgetExceeded) || f.calls != 1 {
				t.Fatalf("unknown usage refunded: calls=%d err=%v", f.calls, err)
			}
		})
	}
}
