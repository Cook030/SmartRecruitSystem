// Package recruitment defines the read-only recruitment query contract.
// It has no dependency on the database, transport, or agent framework.
package recruitment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("record not found in the current HR scope")

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(message string) error { return &ValidationError{Message: message} }

// Shanghai is fixed at UTC+8; it does not depend on the host's local timezone.
var Shanghai = time.FixedZone("Asia/Shanghai", 8*60*60)

type Pagination struct {
	Page     int `json:"page,omitempty" jsonschema:"description=页码，从1开始，默认1，最大1000"`
	PageSize int `json:"page_size,omitempty" jsonschema:"description=每页条数，默认20，最大50"`
}

func (p *Pagination) Normalize() error {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
	if p.Page < 1 || p.Page > 1000 || p.PageSize < 1 || p.PageSize > 50 {
		return invalid("page must be 1..1000 and page_size must be 1..50")
	}
	return nil
}

func (p Pagination) Offset() int { return (p.Page - 1) * p.PageSize }

type DateRange struct {
	StartDate string `json:"start_date,omitempty" jsonschema:"description=投递开始日期 YYYY-MM-DD，上海时区，包含当日"`
	EndDate   string `json:"end_date,omitempty" jsonschema:"description=投递截止日期 YYYY-MM-DD，上海时区，不包含当日；两个日期必须同时提供，最多366天"`
}

func (d DateRange) Bounds() (time.Time, time.Time, error) {
	if d.StartDate == "" && d.EndDate == "" {
		return time.Time{}, time.Time{}, nil
	}
	start, e1 := time.ParseInLocation("2006-01-02", d.StartDate, Shanghai)
	end, e2 := time.ParseInLocation("2006-01-02", d.EndDate, Shanghai)
	if e1 != nil || e2 != nil || !start.Before(end) || end.Sub(start) > 366*24*time.Hour {
		return time.Time{}, time.Time{}, invalid("provide start_date and end_date as YYYY-MM-DD, with 0 < range <= 366 days")
	}
	return start, end, nil
}

type JobQuery struct {
	Pagination
	Keyword  string `json:"keyword,omitempty" jsonschema:"description=岗位标题包含的文字，按字面匹配"`
	Location string `json:"location,omitempty" jsonschema:"description=工作地点包含的文字"`
	Status   *int   `json:"status,omitempty" jsonschema:"description=岗位状态，0下架或1上架，省略则全部"`
}

func (q *JobQuery) Validate() error {
	if err := q.Pagination.Normalize(); err != nil {
		return err
	}
	if q.Status != nil && *q.Status != 0 && *q.Status != 1 {
		return invalid("status must be 0 or 1")
	}
	return textFilters(q.Keyword, q.Location)
}

type CandidateQuery struct {
	Pagination
	DateRange
	JobID        uint64   `json:"job_id,omitempty" jsonschema:"description=岗位ID，省略则查询当前HR所有岗位的投递候选人"`
	Skills       []string `json:"skills,omitempty" jsonschema:"description=必须全部包含的技能文字，最多5项；当前按字符串包含匹配，不是语义检索"`
	EducationMin string   `json:"education_min,omitempty" jsonschema:"description=最低学历，只支持高中、中专、大专、本科、硕士、博士"`
	University   string   `json:"university,omitempty" jsonschema:"description=院校名称包含的文字"`
}

func (q *CandidateQuery) Validate() error {
	if err := q.Pagination.Normalize(); err != nil {
		return err
	}
	if _, _, err := q.Bounds(); err != nil {
		return err
	}
	if _, err := EducationsAtLeast(q.EducationMin); err != nil {
		return err
	}
	if len(q.Skills) > 5 {
		return invalid("at most 5 skills are supported")
	}
	for _, skill := range q.Skills {
		if strings.TrimSpace(skill) == "" {
			return invalid("skills cannot contain blank values")
		}
	}
	return textFilters(append(q.Skills, q.University)...)
}

type CandidateDetailQuery struct {
	CandidateID uint64 `json:"candidate_id" jsonschema:"description=候选人ID，必须来自可访问的候选人或投递查询结果"`
}

func (q *CandidateDetailQuery) Validate() error {
	if q.CandidateID == 0 {
		return invalid("candidate_id is required")
	}
	return nil
}

type ApplicationQuery struct {
	Pagination
	DateRange
	JobID       uint64 `json:"job_id,omitempty" jsonschema:"description=岗位ID，省略则不限制岗位"`
	CandidateID uint64 `json:"candidate_id,omitempty" jsonschema:"description=候选人ID，省略则不限制候选人"`
}

func (q *ApplicationQuery) Validate() error {
	if err := q.Pagination.Normalize(); err != nil {
		return err
	}
	_, _, err := q.Bounds()
	return err
}

type StatsQuery struct {
	DateRange
	JobID   uint64 `json:"job_id,omitempty" jsonschema:"description=可选岗位ID"`
	GroupBy string `json:"group_by,omitempty" jsonschema:"description=summary汇总、job岗位热度、education学历分布，默认summary"`
	Limit   int    `json:"limit,omitempty" jsonschema:"description=分组最多返回条数，默认20，最大50；不影响汇总总数"`
}

func (q *StatsQuery) Validate() error {
	if _, _, err := q.Bounds(); err != nil {
		return err
	}
	if q.GroupBy == "" {
		q.GroupBy = "summary"
	}
	if q.GroupBy != "summary" && q.GroupBy != "job" && q.GroupBy != "education" {
		return invalid("group_by must be summary, job, or education")
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 50 {
		return invalid("limit must be 1..50")
	}
	return nil
}

func textFilters(values ...string) error {
	for _, v := range values {
		if utf8.RuneCountInString(v) > 128 {
			return invalid("filter text cannot exceed 128 characters")
		}
	}
	return nil
}

func EducationsAtLeast(min string) ([]string, error) {
	levels := []string{"高中", "中专", "大专", "本科", "硕士", "博士"}
	if min == "" {
		return nil, nil
	}
	for i, level := range levels {
		if level == min {
			// 高中和中专是同级；未知/自由文本学历不猜测等级。
			if i == 1 {
				i = 0
			}
			return levels[i:], nil
		}
	}
	return nil, invalid(fmt.Sprintf("unsupported education_min: %s", min))
}

// DTOs deliberately exclude phone numbers, passwords, and OSS credentials/URLs.
type Job struct {
	JobID                uint64 `json:"job_id"`
	Title                string `json:"title"`
	Location             string `json:"location"`
	Salary               string `json:"salary"`
	Status               int    `json:"status"`
	Requirement          string `json:"requirement"`
	RequirementTruncated bool   `json:"requirement_truncated,omitempty"`
}

type Candidate struct {
	CandidateID uint64 `json:"candidate_id"`
	Name        string `json:"name"`
	Education   string `json:"education"`
	University  string `json:"university"`
	Skills      string `json:"skills"`
}

type CandidateDetail struct {
	Candidate
	Experience          string `json:"experience"`
	ExperienceTruncated bool   `json:"experience_truncated,omitempty"`
}

type Application struct {
	ApplicationID uint64 `json:"application_id"`
	JobID         uint64 `json:"job_id"`
	JobTitle      string `json:"job_title"`
	CandidateID   uint64 `json:"candidate_id"`
	CandidateName string `json:"candidate_name"`
	Status        int    `json:"status"`
	// Formatted in SQL to preserve Shanghai wall time independent of the DSN location.
	AppliedAt string `json:"applied_at"`
}

type List[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type StatGroup struct {
	JobID        uint64 `json:"job_id,omitempty"`
	Label        string `json:"label"`
	Applications int64  `json:"applications"`
	Candidates   int64  `json:"candidates"`
}

type Stats struct {
	Applications    int64       `json:"applications"`
	Candidates      int64       `json:"candidates"`
	Groups          []StatGroup `json:"groups" gorm:"-"`
	GroupBy         string      `json:"group_by"`
	GroupsTruncated bool        `json:"groups_truncated"`
}

// Reader is the boundary between agent tools and the persistence implementation.
// Identity is supplied separately from model-generated query parameters.
type Reader interface {
	SearchJobs(context.Context, uint64, JobQuery) (*List[Job], error)
	SearchCandidates(context.Context, uint64, CandidateQuery) (*List[Candidate], error)
	GetCandidateDetail(context.Context, uint64, CandidateDetailQuery) (*CandidateDetail, error)
	ListRecruitmentApplications(context.Context, uint64, ApplicationQuery) (*List[Application], error)
	GetRecruitmentStats(context.Context, uint64, StatsQuery) (*Stats, error)
}
