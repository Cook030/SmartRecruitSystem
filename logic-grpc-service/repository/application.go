package repository

import "logic-grpc-service/model"

func (r *Repository) CreateApplication(app *model.Application) error {
	return r.DB.Create(app).Error
}

func (r *Repository) GetApplicationByJobAndCandidate(jobID, candidateID uint64) (*model.Application, error) {
	var app model.Application
	err := r.DB.Where("job_id = ? AND candidate_id = ?", jobID, candidateID).First(&app).Error
	return &app, err
}

func (r *Repository) ListApplicationsByCandidate(candidateID uint64, page, pageSize int) ([]model.Application, int64, error) {
	var apps []model.Application
	var total int64
	query := r.DB.Where("candidate_id = ?", candidateID)
	query.Model(&model.Application{}).Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

func (r *Repository) ListApplicationsByJobID(jobID uint64, page, pageSize int) ([]model.Application, int64, error) {
	var apps []model.Application
	var total int64
	query := r.DB.Where("job_id = ?", jobID)
	query.Model(&model.Application{}).Count(&total)
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&apps).Error
	return apps, total, err
}

func (r *Repository) GetApplicationByID(id uint64) (*model.Application, error) {
	var app model.Application
	err := r.DB.First(&app, id).Error
	return &app, err
}
