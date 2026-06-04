package repository

import "logic-grpc-service/model"

func (r *Repository) CreateResume(resume *model.Resume) error {
	return r.DB.Create(resume).Error
}

func (r *Repository) GetResumeByCandidate(candidateID uint64) (*model.Resume, error) {
	var resume model.Resume
	err := r.DB.Where("candidate_id = ?", candidateID).Order("created_at DESC").First(&resume).Error
	return &resume, err
}

func (r *Repository) GetResumeByID(id uint64) (*model.Resume, error) {
	var resume model.Resume
	err := r.DB.First(&resume, id).Error
	return &resume, err
}

func (r *Repository) DeleteResume(candidateID uint64) error {
	return r.DB.Where("candidate_id = ?", candidateID).Delete(&model.Resume{}).Error
}
