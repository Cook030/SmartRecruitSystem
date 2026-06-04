package model

import "time"

type AsyncTask struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskType     string    `gorm:"type:varchar(64);not null" json:"taskType"`
	BusinessID   uint64    `gorm:"not null;default:0" json:"businessId"`
	UserID       uint64    `gorm:"not null;default:0" json:"userId"`
	UserRole     string    `gorm:"type:varchar(20);not null;default:''" json:"userRole"`
	Status       string    `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Params       string    `gorm:"type:text" json:"params"`
	Result       string    `gorm:"type:text" json:"result"`
	ErrorMessage string    `gorm:"type:text" json:"errorMessage"`
	RetryCount   int       `gorm:"not null;default:0" json:"retryCount"`
	MaxRetries   int       `gorm:"not null;default:3" json:"maxRetries"`
	StartedAt    *time.Time `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (AsyncTask) TableName() string { return "async_tasks" }

type ResumeParseResult struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ResumeID    uint64    `gorm:"uniqueIndex;not null" json:"resumeId"`
	CandidateID uint64    `gorm:"index;not null" json:"candidateId"`
	Education   string    `gorm:"type:text" json:"education"`
	Experience  string    `gorm:"type:text" json:"experience"`
	Projects    string    `gorm:"type:text" json:"projects"`
	Skills      string    `gorm:"type:varchar(500);not null;default:''" json:"skills"`
	RawResult   string    `gorm:"type:text" json:"rawResult"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ResumeParseResult) TableName() string { return "resume_parse_results" }

type AIMatchResult struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	JobID         uint64    `gorm:"index;not null" json:"jobId"`
	CandidateID   uint64    `gorm:"index;not null" json:"candidateId"`
	ApplicationID uint64    `gorm:"uniqueIndex;not null" json:"applicationId"`
	MatchScore    float64   `gorm:"not null;default:0" json:"matchScore"`
	MatchReason   string    `gorm:"type:text" json:"matchReason"`
	Strengths     string    `gorm:"type:text" json:"strengths"`
	Weaknesses    string    `gorm:"type:text" json:"weaknesses"`
	InterviewTips string    `gorm:"type:text" json:"interviewTips"`
	RawResult     string    `gorm:"type:text" json:"rawResult"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (AIMatchResult) TableName() string { return "ai_match_results" }
