package ai

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var ErrTokenBudgetExceeded = errors.New("model token budget exceeded")

const defaultTokenBudget = 128000
const maxOutputTokens = 4096

// A budget belongs to one request, including compaction and all agent steps.
type tokenBudget struct {
	mu        sync.Mutex
	remaining int
	stopped   bool
}

type budgetModel struct {
	inner  model.ToolCallingChatModel
	budget *tokenBudget
	tools  []*schema.ToolInfo
}

func withTokenBudget(inner model.ToolCallingChatModel, limit int) *budgetModel {
	if limit <= 0 {
		limit = defaultTokenBudget
	}
	return &budgetModel{inner: inner, budget: &tokenBudget{remaining: limit}}
}

func (m *budgetModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &budgetModel{inner: inner, budget: m.budget, tools: tools}, nil
}

func (m *budgetModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	options := model.GetCommonOptions(nil, opts...)
	tools := m.tools
	if options.Tools != nil {
		tools = options.Tools
	}
	// Reserve UTF-8 JSON bytes as a conservative text-token estimate, plus
	// envelope overhead and the full output cap. Provider usage settles it.
	data, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	input := len(data) + 1024
	for _, tool := range tools {
		params, err := tool.ParamsOneOf.ToJSONSchema()
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		input += len(tool.Name) + len(tool.Desc) + len(data) + 128
	}
	output := maxOutputTokens
	if options.MaxTokens != nil && *options.MaxTokens > 0 {
		output = min(output, *options.MaxTokens)
	}
	reserved := input + output
	m.budget.mu.Lock()
	if m.budget.stopped || m.budget.remaining < reserved {
		m.budget.stopped = true
		m.budget.mu.Unlock()
		return nil, ErrTokenBudgetExceeded
	}
	m.budget.remaining -= reserved
	m.budget.mu.Unlock()
	response, err := m.inner.Generate(ctx, messages, append(opts, model.WithMaxTokens(output))...)
	// Unknown usage (including errors/timeouts) retains the reservation.
	if response != nil && response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		u := response.ResponseMeta.Usage
		actual := max(u.TotalTokens, u.PromptTokens+u.CompletionTokens)
		if actual > 0 && u.PromptTokens >= 0 && u.CompletionTokens >= 0 {
			m.budget.mu.Lock()
			m.budget.remaining += reserved - actual
			if m.budget.remaining <= 0 {
				m.budget.stopped = true
			}
			m.budget.mu.Unlock()
		}
	}
	m.budget.mu.Lock()
	exceeded := m.budget.remaining < 0
	m.budget.mu.Unlock()
	if exceeded {
		return response, ErrTokenBudgetExceeded
	}
	return response, err
}

// Current callers use Generate. Reject streaming rather than bypass accounting.
func (m *budgetModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("streaming token accounting is not supported")
}
