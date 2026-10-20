package repository

import (
	"context"
	"errors"
	"time"

	"logic-grpc-service/recruitment"

	"gorm.io/gorm"
)

const recruitmentQueryTimeout = 3 * time.Second

// Every public query validates again so callers other than the agent cannot
// bypass pagination or identity checks. No SQL identifiers come from the model.
func (r *Repository) SearchJobs(ctx context.Context, hrID uint64, q recruitment.JobQuery) (*recruitment.List[recruitment.Job], error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, recruitmentQueryTimeout)
	defer cancel()
	db, err := r.recruitmentDB(ctx, hrID)
	if err != nil {
		return nil, err
	}
	query := db.Table("jobs").Where("jobs.hr_id = ?", hrID)
	if q.Keyword != "" {
		query = query.Where("LOCATE(?, jobs.title) > 0", q.Keyword)
	}
	if q.Location != "" {
		query = query.Where("LOCATE(?, jobs.location) > 0", q.Location)
	}
	if q.Status != nil {
		query = query.Where("jobs.status = ?", *q.Status)
	}
	result := &recruitment.List[recruitment.Job]{Items: []recruitment.Job{}, Page: q.Page, PageSize: q.PageSize}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	err = query.Select("jobs.id AS job_id, jobs.title, jobs.location, jobs.salary, jobs.status, LEFT(COALESCE(jobs.requirement, ''), 2000) AS requirement, CHAR_LENGTH(COALESCE(jobs.requirement, '')) > 2000 AS requirement_truncated").
		Order("jobs.id DESC").Offset(q.Offset()).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}

func (r *Repository) SearchCandidates(ctx context.Context, hrID uint64, q recruitment.CandidateQuery) (*recruitment.List[recruitment.Candidate], error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, recruitmentQueryTimeout)
	defer cancel()
	db, err := r.recruitmentDB(ctx, hrID)
	if err != nil {
		return nil, err
	}
	// EXISTS keeps one row per profile even if the candidate applied to many jobs.
	query := db.Table("candidate_profiles AS cp").Where("EXISTS (?)", visibleApplications(db, hrID, q.JobID, q.DateRange).Select("1").Where("applications.candidate_id = cp.candidate_id"))
	for _, skill := range q.Skills {
		query = query.Where("LOCATE(?, cp.skills) > 0", skill)
	}
	if q.University != "" {
		query = query.Where("LOCATE(?, cp.university) > 0", q.University)
	}
	if q.EducationMin != "" {
		levels, _ := recruitment.EducationsAtLeast(q.EducationMin)
		query = query.Where("cp.education IN ?", levels)
	}
	result := &recruitment.List[recruitment.Candidate]{Items: []recruitment.Candidate{}, Page: q.Page, PageSize: q.PageSize}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	err = query.Select("cp.candidate_id, cp.name, cp.education, cp.university, cp.skills").
		Order("cp.candidate_id DESC").Offset(q.Offset()).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}

func (r *Repository) GetCandidateDetail(ctx context.Context, hrID uint64, q recruitment.CandidateDetailQuery) (*recruitment.CandidateDetail, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, recruitmentQueryTimeout)
	defer cancel()
	db, err := r.recruitmentDB(ctx, hrID)
	if err != nil {
		return nil, err
	}
	var result recruitment.CandidateDetail
	err = db.Table("candidate_profiles AS cp").
		Where("cp.candidate_id = ?", q.CandidateID).
		Where("EXISTS (?)", visibleApplications(db, hrID, 0, recruitment.DateRange{}).Select("1").Where("applications.candidate_id = cp.candidate_id")).
		Select("cp.candidate_id, cp.name, cp.education, cp.university, cp.skills, LEFT(COALESCE(cp.experience, ''), 4000) AS experience, CHAR_LENGTH(COALESCE(cp.experience, '')) > 4000 AS experience_truncated").Take(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, recruitment.ErrNotFound
	}
	return &result, err
}

func (r *Repository) ListRecruitmentApplications(ctx context.Context, hrID uint64, q recruitment.ApplicationQuery) (*recruitment.List[recruitment.Application], error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, recruitmentQueryTimeout)
	defer cancel()
	db, err := r.recruitmentDB(ctx, hrID)
	if err != nil {
		return nil, err
	}
	query := visibleApplications(db, hrID, q.JobID, q.DateRange).
		Joins("LEFT JOIN candidate_profiles AS cp ON cp.candidate_id = applications.candidate_id")
	if q.CandidateID != 0 {
		query = query.Where("applications.candidate_id = ?", q.CandidateID)
	}
	result := &recruitment.List[recruitment.Application]{Items: []recruitment.Application{}, Page: q.Page, PageSize: q.PageSize}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	err = query.Select("applications.id AS application_id, applications.job_id, jobs.title AS job_title, applications.candidate_id, COALESCE(cp.name, '') AS candidate_name, applications.status, DATE_FORMAT(applications.created_at, '%Y-%m-%d %H:%i:%s') AS applied_at").
		Order("applications.created_at DESC, applications.id DESC").Offset(q.Offset()).Limit(q.PageSize).Scan(&result.Items).Error
	return result, err
}

func (r *Repository) GetRecruitmentStats(ctx context.Context, hrID uint64, q recruitment.StatsQuery) (*recruitment.Stats, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, recruitmentQueryTimeout)
	defer cancel()
	db, err := r.recruitmentDB(ctx, hrID)
	if err != nil {
		return nil, err
	}
	result := &recruitment.Stats{Groups: []recruitment.StatGroup{}, GroupBy: q.GroupBy}
	base := visibleApplications(db, hrID, q.JobID, q.DateRange)
	if err := base.Select("COUNT(*) AS applications, COUNT(DISTINCT applications.candidate_id) AS candidates").Scan(result).Error; err != nil {
		return nil, err
	}
	if q.GroupBy == "summary" {
		return result, nil
	}
	var query *gorm.DB
	if q.GroupBy == "job" {
		// Date filters live inside the joined subquery, preserving jobs with zero applications.
		apps := applicationDates(db.Table("applications"), q.DateRange)
		query = db.Table("jobs").Joins("LEFT JOIN (?) AS a ON a.job_id = jobs.id", apps).
			Where("jobs.hr_id = ?", hrID).
			Select("jobs.id AS job_id, jobs.title AS label, COUNT(a.id) AS applications, COUNT(DISTINCT a.candidate_id) AS candidates").
			Group("jobs.id, jobs.title").Order("applications DESC, jobs.id ASC")
		if q.JobID != 0 {
			query = query.Where("jobs.id = ?", q.JobID)
		}
	} else {
		query = visibleApplications(db, hrID, q.JobID, q.DateRange).
			Joins("LEFT JOIN candidate_profiles AS cp ON cp.candidate_id = applications.candidate_id").
			Select("COALESCE(NULLIF(cp.education, ''), '未知') AS label, COUNT(*) AS applications, COUNT(DISTINCT applications.candidate_id) AS candidates").
			Group("COALESCE(NULLIF(cp.education, ''), '未知')").Order("candidates DESC, label ASC")
	}
	if err := query.Limit(q.Limit + 1).Scan(&result.Groups).Error; err != nil {
		return nil, err
	}
	if len(result.Groups) > q.Limit {
		result.Groups = result.Groups[:q.Limit]
		result.GroupsTruncated = true
	}
	return result, nil
}

func (r *Repository) recruitmentDB(ctx context.Context, hrID uint64) (*gorm.DB, error) {
	if hrID == 0 {
		return nil, &recruitment.ValidationError{Message: "authenticated HR identity is required"}
	}
	return r.DB.WithContext(ctx), nil
}

func visibleApplications(db *gorm.DB, hrID, jobID uint64, dates recruitment.DateRange) *gorm.DB {
	query := db.Table("applications").Joins("JOIN jobs ON jobs.id = applications.job_id").Where("jobs.hr_id = ?", hrID)
	if jobID != 0 {
		query = query.Where("applications.job_id = ?", jobID)
	}
	return applicationDates(query, dates)
}

func applicationDates(query *gorm.DB, dates recruitment.DateRange) *gorm.DB {
	// Public entry points validate dates first. Bind wall-clock strings because
	// MySQL DATETIME has no timezone and the legacy service stores local wall time.
	if dates.StartDate != "" {
		query = query.Where("applications.created_at >= ? AND applications.created_at < ?", dates.StartDate+" 00:00:00", dates.EndDate+" 00:00:00")
	}
	return query
}
