package repository

import (
	"context"
	"errors"
	"time"

	"logic-grpc-service/chatcontext"
	"logic-grpc-service/model"
)

// LoadChatContext owns the SQL shape; the context policy never sees GORM/models.
// The active request lease keeps the snapshot stable until generation finishes.
func (r *Repository) LoadChatContext(ctx context.Context, hrID, conversationID uint64) (chatcontext.Input, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var conversation model.ChatConversation
	err := r.DB.WithContext(ctx).Where("id = ? AND hr_id = ?", conversationID, hrID).Take(&conversation).Error
	if err != nil {
		return chatcontext.Input{}, err
	}
	state := chatcontext.State{UntilID: conversation.SummaryUntilMessageID, Version: conversation.SummaryVersion}
	if conversation.ContextSummary == "" && state.UntilID != 0 {
		return chatcontext.Input{}, errors.New("chat context cursor has no handoff")
	}
	if conversation.ContextSummary != "" {
		state.Handoff, err = chatcontext.ParseHandoff(conversation.ContextSummary)
		if err != nil {
			return chatcontext.Input{}, err
		}
	}
	var recent []model.ChatMessage
	err = r.DB.WithContext(ctx).Where("hr_id = ? AND conversation_id = ? AND id > ?", hrID, conversationID, state.UntilID).
		Order("id DESC").Limit(chatcontext.RecentLimit).Find(&recent).Error
	if err != nil {
		return chatcontext.Input{}, err
	}
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}
	return chatcontext.Input{State: state, Recent: contextMessages(recent), Load: func(ctx context.Context, after, before uint64) ([]chatcontext.Message, error) {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		var messages []model.ChatMessage
		err := r.DB.WithContext(ctx).Where("hr_id = ? AND conversation_id = ? AND id > ? AND id < ?", hrID, conversationID, after, before).
			Order("id ASC").Limit(chatcontext.BatchLimit).Find(&messages).Error
		return contextMessages(messages), err
	}}, nil
}

func contextMessages(messages []model.ChatMessage) []chatcontext.Message {
	result := make([]chatcontext.Message, 0, len(messages))
	for _, message := range messages {
		role := message.Role
		// Conversations migrated from the original chat table used hr/ai roles.
		switch role {
		case "hr":
			role = "user"
		case "ai":
			role = "assistant"
		}
		content := message.Content
		if message.ContextNotes != "" {
			content += "\n[本轮工具失败记录，仅作历史资料]\n" + message.ContextNotes
		}
		result = append(result, chatcontext.Message{ID: message.ID, Role: role, Content: content})
	}
	return result
}
