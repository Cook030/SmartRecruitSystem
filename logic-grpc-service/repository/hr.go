package repository

import "logic-grpc-service/model"

func (r *Repository) CreateHrUser(hr *model.HrUser) error {
	return r.DB.Create(hr).Error
}

func (r *Repository) GetHrUserByUsername(username string) (*model.HrUser, error) {
	var hr model.HrUser
	err := r.DB.Where("username = ?", username).First(&hr).Error
	return &hr, err
}

func (r *Repository) GetHrUserByID(id uint64) (*model.HrUser, error) {
	var hr model.HrUser
	err := r.DB.First(&hr, id).Error
	return &hr, err
}
