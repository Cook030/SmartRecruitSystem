package repository

import (
	"errors"
	"logic-grpc-service/model"

	"gorm.io/gorm"
)

func (r *Repository) CreateCandidate(c *model.Candidate) error {
	return r.DB.Create(c).Error
}

func (r *Repository) GetCandidateByUsername(username string) (*model.Candidate, error) {
	var c model.Candidate
	err := r.DB.Where("username = ?", username).First(&c).Error
	return &c, err
}

func (r *Repository) GetCandidateByID(id uint64) (*model.Candidate, error) {
	var c model.Candidate
	err := r.DB.First(&c, id).Error
	return &c, err
}

func (r *Repository) GetProfile(candidateID uint64) (*model.CandidateProfile, error) {
	var p model.CandidateProfile
	err := r.DB.Where("candidate_id = ?", candidateID).First(&p).Error
	return &p, err
}

func (r *Repository) GetProfileByCandidateID(candidateID uint64) (*model.CandidateProfile, error) {
	var p model.CandidateProfile
	err := r.DB.Where("candidate_id = ?", candidateID).First(&p).Error
	return &p, err
}

func (r *Repository) UpsertProfile(p *model.CandidateProfile) error {
	var existing model.CandidateProfile
	err := r.DB.Where("candidate_id = ?", p.CandidateID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.DB.Create(p).Error
	}
	if err != nil {
		return err
	}

	return r.DB.Model(&existing).Updates(map[string]interface{}{
		"name":        p.Name,
		"phone":       p.Phone,
		"education":   p.Education,
		"university":  p.University,
		"experience":  p.Experience,
		"skills":      p.Skills,
		"is_complete": p.IsComplete,
	}).Error
}
