package model

import "time"

type Resume struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CandidateID uint64   `gorm:"index;not null" json:"candidateId"`
	FileName    string    `gorm:"type:varchar(255);not null" json:"fileName"`
	OssKey      string    `gorm:"type:varchar(500);not null" json:"ossKey"`
	FileSize    int64     `gorm:"not null;default:0" json:"fileSize"`
	FileType    string    `gorm:"type:varchar(20);not null" json:"fileType"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Resume) TableName() string { return "resumes" }
