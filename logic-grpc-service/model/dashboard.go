package model

import "time"

type DashboardStat struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HrID      uint64    `gorm:"not null;default:0" json:"hrId"`
	StatKey   string    `gorm:"type:varchar(128);not null" json:"statKey"`
	StatValue string    `gorm:"type:text;not null" json:"statValue"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (DashboardStat) TableName() string { return "dashboard_stats" }
