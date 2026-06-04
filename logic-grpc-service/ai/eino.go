package ai

import (
	"context"
	"fmt"

	"logic-grpc-service/config"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func Chat(cfg *config.Config, history []*schema.Message, userMessage string) (string, error) {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.AI.BaseURL,
		Model:   cfg.AI.Model,
		APIKey:  cfg.AI.APIKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat model: %w", err)
	}

	messages := make([]*schema.Message, 0, len(history)+2)
	messages = append(messages, &schema.Message{
		Role: schema.System,
		Content: `你是一个智能招聘系统的 AI 助手，服务于 HR 用户。你拥有系统中 MySQL 数据库的真实业务数据，包括：
1. 岗位信息（名称、地点、薪资、投递人数）
2. 候选人档案（姓名、学历、毕业院校、技能标签、工作/项目经历）
3. 投递关系（哪位候选人投递了哪个岗位）
4. 学历分布统计

你可以回答以下类型的问题：
- 投递人数统计（总数、单岗位投递数）
- 岗位热度分析（哪个岗位最热门/最冷门）
- 候选人筛选（按技能、学历、院校等条件筛选合适的候选人）
- 候选人推荐（根据岗位要求推荐匹配的候选人）
- 学历分布分析
- 候选人与岗位的匹配度分析

请严格基于提供的真实数据回答，不要编造不存在的数据。用简洁专业的中文回答。`,
	})
	messages = append(messages, history...)
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userMessage,
	})

	resp, err := chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate: %w", err)
	}

	return resp.Content, nil
}
