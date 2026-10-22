package ai

import (
	"context"
	"fmt"

	"logic-grpc-service/config"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func CallAI(cfg *config.Config, prompt string) (string, error) {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.AI.BaseURL,
		Model:   cfg.AI.Model,
		APIKey:  cfg.AI.APIKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat model: %w", err)
	}

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个智能招聘系统的 AI 分析助手。请严格按照用户要求的格式返回结果，不要添加多余的解释。",
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	resp, err := withTokenBudget(withModelCircuit(chatModel), cfg.AI.TokenBudget).Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate: %w", err)
	}

	return resp.Content, nil
}
