package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"logic-grpc-service/chatcontext"
	"logic-grpc-service/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrChatBusy           = errors.New("conversation has an active request")
	ErrChatConflict       = errors.New("request ID was already used for different content or conversation")
	ErrChatAttemptExpired = errors.New("chat generation lease expired")
)

func (r *Repository) CreateChatConversation(ctx context.Context, hrID uint64, title string) (*model.ChatConversation, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conversation := &model.ChatConversation{HrID: hrID, Title: title}
	err := r.DB.WithContext(ctx).Create(conversation).Error
	return conversation, err
}

func (r *Repository) ListChatConversations(ctx context.Context, hrID, beforeID uint64) ([]model.ChatConversation, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var conversations []model.ChatConversation
	q := r.DB.WithContext(ctx).Where("hr_id = ?", hrID)
	if beforeID != 0 {
		q = q.Where("id < ?", beforeID)
	}
	err := q.Order("id DESC").Limit(51).Find(&conversations).Error
	return conversations, err
}

func lockChatConversation(tx *gorm.DB, hrID, conversationID uint64) error {
	var conversation model.ChatConversation
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND hr_id = ?", conversationID, hrID).Take(&conversation).Error
}

// The conversation row serializes admission across replicas. The transaction
// ends before any model call; a 120s lease outlives the 90s generation deadline.
func (r *Repository) BeginChatRequest(ctx context.Context, hrID, conversationID uint64, requestID, content string) (*model.ChatRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var request model.ChatRequest
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockChatConversation(tx, hrID, conversationID); err != nil {
			return err
		}
		err := tx.Where("hr_id = ? AND request_id = ?", hrID, requestID).Take(&request).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if request.ConversationID != conversationID || request.Content != content {
				return ErrChatConflict
			}
			if request.Status == "completed" {
				return nil
			}
		}
		var active int64
		if err := tx.Model(&model.ChatRequest{}).Where("conversation_id = ? AND status = 'pending' AND lease_until > NOW(6)", conversationID).Count(&active).Error; err != nil {
			return err
		}
		if active != 0 {
			return ErrChatBusy
		}
		if err := tx.Model(&model.ChatRequest{}).Where("conversation_id = ? AND status = 'pending'", conversationID).Update("status", "failed").Error; err != nil {
			return err
		}
		request.HrID, request.ConversationID = hrID, conversationID
		request.RequestID, request.Content = requestID, content
		request.Status, request.Attempt = "pending", rand.Text()
		if request.ID == 0 {
			request.LeaseUntil = time.Now().Add(2 * time.Minute)
			if err := tx.Create(&request).Error; err != nil {
				return err
			}
		}
		// An explicit DB clock avoids lease drift between service instances.
		return tx.Model(&request).Updates(map[string]any{"status": "pending", "attempt": request.Attempt, "lease_until": gorm.Expr("DATE_ADD(NOW(6), INTERVAL 120 SECOND)")}).Error
	})
	return &request, err
}

// Reply, complete message pair and request state commit atomically. An expired
// worker cannot commit after a retry or a newer turn has taken over.
func (r *Repository) CompleteChatRequest(ctx context.Context, request *model.ChatRequest, reply, notes string, update *chatcontext.State) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockChatConversation(tx, request.HrID, request.ConversationID); err != nil {
			return err
		}
		result := tx.Model(&model.ChatRequest{}).Where("id = ? AND attempt = ? AND status = 'pending' AND lease_until > NOW(6)", request.ID, request.Attempt).
			Updates(map[string]any{"status": "completed", "reply": reply})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrChatAttemptExpired
		}
		messages := []model.ChatMessage{
			{HrID: request.HrID, ConversationID: request.ConversationID, Role: "user", Content: request.Content},
			{HrID: request.HrID, ConversationID: request.ConversationID, Role: "assistant", Content: reply, ContextNotes: notes},
		}
		if err := tx.Create(&messages).Error; err != nil {
			return err
		}
		q := tx.Model(&model.ChatConversation{}).Where("id = ? AND hr_id = ?", request.ConversationID, request.HrID)
		values := map[string]any{"updated_at": time.Now()}
		if update != nil {
			if update.Version == 0 || update.Handoff == nil {
				return ErrChatConflict
			}
			raw := update.Handoff.JSON()
			if _, err := chatcontext.ParseHandoff(raw); err != nil {
				return err
			}
			q = q.Where("summary_version = ? AND summary_until_message_id < ?", update.Version-1, update.UntilID)
			values["context_summary"], values["summary_until_message_id"], values["summary_version"] = raw, update.UntilID, update.Version
		}
		result = q.Updates(values)
		if result.Error != nil {
			return result.Error
		}
		if update != nil && result.RowsAffected != 1 {
			return ErrChatConflict
		}
		return nil
	})
}

func (r *Repository) FailChatRequest(ctx context.Context, request *model.ChatRequest) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return r.DB.WithContext(ctx).Model(&model.ChatRequest{}).
		Where("id = ? AND attempt = ? AND status = 'pending'", request.ID, request.Attempt).Update("status", "failed").Error
}

func (r *Repository) GetChatHistory(ctx context.Context, hrID, conversationID, beforeID uint64, limit int) ([]model.ChatMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var conversation model.ChatConversation
	if err := r.DB.WithContext(ctx).Where("id = ? AND hr_id = ?", conversationID, hrID).Take(&conversation).Error; err != nil {
		return nil, err
	}
	if limit < 1 || limit > 101 {
		limit = 40
	}
	q := r.DB.WithContext(ctx).Where("hr_id = ? AND conversation_id = ?", hrID, conversationID)
	if beforeID != 0 {
		q = q.Where("id < ?", beforeID)
	}
	var messages []model.ChatMessage
	err := q.Order("id DESC").Limit(limit).Find(&messages).Error
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, err
}

func (r *Repository) DeleteChatConversation(ctx context.Context, hrID, conversationID uint64) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockChatConversation(tx, hrID, conversationID); err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&model.ChatRequest{}).Where("conversation_id = ? AND status = 'pending' AND lease_until > NOW(6)", conversationID).Count(&active).Error; err != nil {
			return err
		}
		if active != 0 {
			return ErrChatBusy
		}
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.ChatRequest{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ChatConversation{}, conversationID).Error
	})
}
