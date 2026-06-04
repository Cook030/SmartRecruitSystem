package repository

import "logic-grpc-service/model"

func (r *Repository) CreateChatMessage(msg *model.ChatMessage) error {
	return r.DB.Create(msg).Error
}

func (r *Repository) GetChatHistory(hrID uint64) ([]model.ChatMessage, error) {
	var msgs []model.ChatMessage
	err := r.DB.Where("hr_id = ?", hrID).Order("created_at ASC").Find(&msgs).Error
	return msgs, err
}
