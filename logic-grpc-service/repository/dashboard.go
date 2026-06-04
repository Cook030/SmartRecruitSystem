package repository

import (
	"time"

	"logic-grpc-service/model"
)

type CountPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type HotJobResult struct {
	JobID            uint64 `json:"job_id"`
	Title            string `json:"title"`
	Location         string `json:"location"`
	Salary           string `json:"salary"`
	ApplicationCount int64  `json:"application_count"`
}

func (r *Repository) CountJobsByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Job{}).Where("hr_id = ?", hrID).Count(&count).Error
	return count, err
}

func (r *Repository) CountOnlineJobsByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Job{}).Where("hr_id = ? AND status = ?", hrID, 1).Count(&count).Error
	return count, err
}

func (r *Repository) CountOfflineJobsByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Job{}).Where("hr_id = ? AND status = ?", hrID, 0).Count(&count).Error
	return count, err
}

func (r *Repository) CountApplicationsByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Table("applications").Joins("JOIN jobs ON jobs.id = applications.job_id").Where("jobs.hr_id = ?", hrID).Count(&count).Error
	return count, err
}

func (r *Repository) CountCandidates() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Candidate{}).Count(&count).Error
	return count, err
}

func (r *Repository) CountResumes() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Resume{}).Count(&count).Error
	return count, err
}

func (r *Repository) CountResumesByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Table("resumes").Joins("JOIN candidates ON candidates.id = resumes.candidate_id").Joins("JOIN applications ON applications.candidate_id = candidates.id").Joins("JOIN jobs ON jobs.id = applications.job_id").Where("jobs.hr_id = ?", hrID).Distinct("resumes.id").Count(&count).Error
	return count, err
}

func (r *Repository) CountChatsByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.ChatMessage{}).Where("hr_id = ?", hrID).Count(&count).Error
	return count, err
}

func (r *Repository) CountChatMessagesByHrID(hrID uint64) (int64, int64, int64, error) {
	var total, hrMsgs, aiMsgs int64
	if err := r.DB.Model(&model.ChatMessage{}).Where("hr_id = ?", hrID).Count(&total).Error; err != nil {
		return 0, 0, 0, err
	}
	if err := r.DB.Model(&model.ChatMessage{}).Where("hr_id = ? AND role = ?", hrID, "hr").Count(&hrMsgs).Error; err != nil {
		return 0, 0, 0, err
	}
	if err := r.DB.Model(&model.ChatMessage{}).Where("hr_id = ? AND role = ?", hrID, "ai").Count(&aiMsgs).Error; err != nil {
		return 0, 0, 0, err
	}
	return total, hrMsgs, aiMsgs, nil
}

func (r *Repository) GetHotJobsByHrID(hrID uint64) ([]HotJobResult, error) {
	var results []HotJobResult
	err := r.DB.Table("jobs").
		Select("jobs.id as job_id, jobs.title, jobs.location, jobs.salary, COUNT(applications.id) as application_count").
		Joins("LEFT JOIN applications ON applications.job_id = jobs.id").
		Where("jobs.hr_id = ?", hrID).
		Group("jobs.id").
		Order("application_count DESC").
		Find(&results).Error
	return results, err
}

func (r *Repository) GetCandidateProfilesWithSkills() ([]model.CandidateProfile, error) {
	var profiles []model.CandidateProfile
	err := r.DB.Where("is_complete = ?", 1).Find(&profiles).Error
	return profiles, err
}

func (r *Repository) GetJobsCreatedBetween(hrID uint64, start, end time.Time) ([]CountPoint, error) {
	var results []CountPoint
	err := r.DB.Model(&model.Job{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("hr_id = ? AND created_at BETWEEN ? AND ?", hrID, start, end).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&results).Error
	return results, err
}

func (r *Repository) GetApplicationsCreatedBetween(hrID uint64, start, end time.Time) ([]CountPoint, error) {
	var results []CountPoint
	err := r.DB.Table("applications").
		Select("DATE(applications.created_at) as date, COUNT(*) as count").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND applications.created_at BETWEEN ? AND ?", hrID, start, end).
		Group("DATE(applications.created_at)").
		Order("date ASC").
		Find(&results).Error
	return results, err
}

func (r *Repository) CountCandidatesByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Table("candidates").
		Joins("JOIN applications ON applications.candidate_id = candidates.id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID).
		Distinct("candidates.id").
		Count(&count).Error
	return count, err
}

func (r *Repository) CountCompleteProfilesByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Table("candidate_profiles").
		Joins("JOIN applications ON applications.candidate_id = candidate_profiles.candidate_id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND candidate_profiles.is_complete = ?", hrID, 1).
		Distinct("candidate_profiles.candidate_id").
		Count(&count).Error
	return count, err
}

func (r *Repository) CountResumesUploadedByHrID(hrID uint64) (int64, error) {
	var count int64
	err := r.DB.Table("resumes").
		Joins("JOIN applications ON applications.candidate_id = resumes.candidate_id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ?", hrID).
		Distinct("resumes.candidate_id").
		Count(&count).Error
	return count, err
}

func (r *Repository) GetCandidateProfilesWithSkillsByHrID(hrID uint64) ([]model.CandidateProfile, error) {
	var profiles []model.CandidateProfile
	err := r.DB.Joins("JOIN applications ON applications.candidate_id = candidate_profiles.candidate_id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND candidate_profiles.is_complete = ?", hrID, 1).
		Distinct("candidate_profiles.candidate_id").
		Find(&profiles).Error
	return profiles, err
}

func (r *Repository) GetAllCandidateProfilesWithSkillsByHrID(hrID uint64) ([]model.CandidateProfile, error) {
	var profiles []model.CandidateProfile
	err := r.DB.Joins("JOIN applications ON applications.candidate_id = candidate_profiles.candidate_id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.hr_id = ? AND candidate_profiles.skills != ''", hrID).
		Distinct("candidate_profiles.candidate_id").
		Find(&profiles).Error
	return profiles, err
}
