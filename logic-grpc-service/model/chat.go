package model

import "time"

type ChatMessage struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HrID      uint64    `gorm:"index;not null" json:"hrId"`
	Role      string    `gorm:"type:varchar(20);not null" json:"role"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChatMessage) TableName() string { return "chat_messages" }
