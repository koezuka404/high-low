package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type IUserRepository interface {
	Create(user *model.User) error
	GetUserByEmail(user *model.User, email string) error
	GetUserByID(user *model.User, id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetUserByEmail(user *model.User, email string) error {
	return r.db.Where("email = ?", email).First(user).Error
}

func (r *userRepository) GetUserByID(user *model.User, id uint) error {
	return r.db.First(user, id).Error
}
