package repository

import "logic-grpc-service/model"

func (r *Repository) CreateJob(job *model.Job) error {
	return r.DB.Create(job).Error
}

func (r *Repository) UpdateJob(job *model.Job) error {
	return r.DB.Save(job).Error
}

func (r *Repository) GetJobByID(id uint64) (*model.Job, error) {
	var job model.Job
	err := r.DB.First(&job, id).Error
	return &job, err
}

func (r *Repository) ListJobsByHrID(hrID uint64, page, pageSize int) ([]model.Job, int64, error) {
	var jobs []model.Job
	var total int64
	query := r.DB.Where("hr_id = ?", hrID)
	query.Model(&model.Job{}).Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&jobs).Error
	return jobs, total, err
}

func (r *Repository) ListPublicJobs(page, pageSize int) ([]model.Job, int64, error) {
	var jobs []model.Job
	var total int64
	query := r.DB.Where("status = ?", 1)
	query.Model(&model.Job{}).Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&jobs).Error
	return jobs, total, err
}

func (r *Repository) OfflineJob(id, hrID uint64) error {
	result := r.DB.Model(&model.Job{}).Where("id = ? AND hr_id = ?", id, hrID).Update("status", 0)
	if result.RowsAffected == 0 {
		return result.Error
	}
	return result.Error
}

// AI 统计查询
func (r *Repository) CountAllApplications() (int64, error) {
	var count int64
	err := r.DB.Model(&model.Application{}).Count(&count).Error
	return count, err
}

func (r *Repository) CountApplicationsByJobID(jobID uint64) (int64, error) {
	var count int64
	err := r.DB.Model(&model.Application{}).Where("job_id = ?", jobID).Count(&count).Error
	return count, err
}

func (r *Repository) GetAllJobsWithApplicationCount() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.DB.Model(&model.Job{}).
		Select("jobs.id, jobs.title, jobs.location, jobs.salary, COUNT(applications.id) as application_count").
		Joins("LEFT JOIN applications ON applications.job_id = jobs.id").
		Group("jobs.id").
		Order("application_count DESC").
		Find(&results).Error
	return results, err
}

func (r *Repository) SearchCandidatesBySkills(skills string) ([]model.CandidateProfile, error) {
	var profiles []model.CandidateProfile
	err := r.DB.Where("skills LIKE ?", "%"+skills+"%").Find(&profiles).Error
	return profiles, err
}

func (r *Repository) GetAllCandidateProfiles() ([]model.CandidateProfile, error) {
	var profiles []model.CandidateProfile
	err := r.DB.Where("is_complete = ?", 1).Find(&profiles).Error
	return profiles, err
}

type CandidateApplicationDetail struct {
	CandidateName string
	Education     string
	University    string
	Skills        string
	JobTitle      string
}

func (r *Repository) GetCandidateApplicationDetails() ([]CandidateApplicationDetail, error) {
	var results []CandidateApplicationDetail
	err := r.DB.Table("applications").
		Select("candidate_profiles.name as candidate_name, candidate_profiles.education, candidate_profiles.university, candidate_profiles.skills, jobs.title as job_title").
		Joins("JOIN candidate_profiles ON candidate_profiles.candidate_id = applications.candidate_id").
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Find(&results).Error
	return results, err
}

func (r *Repository) CountCandidateProfiles() (int64, error) {
	var count int64
	err := r.DB.Model(&model.CandidateProfile{}).Where("is_complete = ?", 1).Count(&count).Error
	return count, err
}

func (r *Repository) GetEducationDistribution() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.DB.Model(&model.CandidateProfile{}).
		Select("education, COUNT(*) as count").
		Where("is_complete = ? AND education != ''", 1).
		Group("education").
		Find(&results).Error
	return results, err
}
