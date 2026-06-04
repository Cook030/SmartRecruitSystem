package model

import "time"

type Application struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	JobID       uint64    `gorm:"not null" json:"jobId"`
	CandidateID uint64   `gorm:"not null" json:"candidateId"`
	Status      int8     `gorm:"type:tinyint;not null;default:0" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Application) TableName() string { return "applications" }
