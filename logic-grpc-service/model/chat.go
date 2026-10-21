package model

import "time"

type ChatMessage struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HrID           uint64    `gorm:"index;not null" json:"hrId"`
	ConversationID uint64    `gorm:"index;not null" json:"conversationId"`
	Role           string    `gorm:"type:varchar(20);not null" json:"role"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	ContextNotes   string    `gorm:"type:text" json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (ChatMessage) TableName() string { return "chat_messages" }

type ChatConversation struct {
	ID                    uint64 `gorm:"primaryKey;autoIncrement"`
	HrID                  uint64 `gorm:"index;not null"`
	Title                 string `gorm:"size:80;not null"`
	ContextSummary        string `gorm:"type:text"`
	SummaryUntilMessageID uint64 `gorm:"not null;default:0"`
	SummaryVersion        uint64 `gorm:"not null;default:0"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (ChatConversation) TableName() string { return "chat_conversations" }

// A request survives transport failures so a retry can recover a committed reply.
type ChatRequest struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	HrID           uint64    `gorm:"uniqueIndex:uk_chat_request,priority:1;not null"`
	RequestID      string    `gorm:"size:64;uniqueIndex:uk_chat_request,priority:2;not null"`
	ConversationID uint64    `gorm:"index;not null"`
	Content        string    `gorm:"type:text;not null"`
	Reply          string    `gorm:"type:longtext;not null"`
	Status         string    `gorm:"size:20;not null"`
	Attempt        string    `gorm:"size:64;not null"`
	LeaseUntil     time.Time `gorm:"precision:6;not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (ChatRequest) TableName() string { return "chat_requests" }
