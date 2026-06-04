package model

import "time"

type Job struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HrID        uint64    `gorm:"index;not null" json:"hrId"`
	Title       string    `gorm:"type:varchar(128);not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Requirement string    `gorm:"type:text" json:"requirement"`
	Salary      string    `gorm:"type:varchar(64);not null;default:''" json:"salary"`
	Location    string    `gorm:"type:varchar(128);not null;default:''" json:"location"`
	Status      int8      `gorm:"type:tinyint(1);not null;default:1" json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (Job) TableName() string { return "jobs" }
