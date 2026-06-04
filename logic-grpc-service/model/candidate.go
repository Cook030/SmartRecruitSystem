package model

import "time"

type Candidate struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Candidate) TableName() string { return "candidates" }

type CandidateProfile struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CandidateID uint64   `gorm:"uniqueIndex;not null" json:"candidateId"`
	Name        string   `gorm:"type:varchar(64);not null;default:''" json:"name"`
	Phone       string   `gorm:"type:varchar(20);not null;default:''" json:"phone"`
	Education   string   `gorm:"type:varchar(32);not null;default:''" json:"education"`
	University  string   `gorm:"type:varchar(128);not null;default:''" json:"university"`
	Experience  string   `gorm:"type:text" json:"experience"`
	Skills      string   `gorm:"type:varchar(500);not null;default:''" json:"skills"`
	IsComplete  bool     `gorm:"type:tinyint(1);not null;default:0" json:"isComplete"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (CandidateProfile) TableName() string { return "candidate_profiles" }
