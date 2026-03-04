package repository

import (
	"time"

	"backend/model"

	"gorm.io/gorm"
)

type IUserSessionRepository interface {
	Create(session *model.UserSession) error
	FindByID(session *model.UserSession, id string) error
	Delete(id string) error
	RefreshTTL(id string, expiresAt time.Time) error
}

type userSessionRepository struct {
	db *gorm.DB
}

func NewUserSessionRepository(db *gorm.DB) IUserSessionRepository {
	return &userSessionRepository{db: db}
}

func (r *userSessionRepository) Create(session *model.UserSession) error {
	return r.db.Create(session).Error
}

func (r *userSessionRepository) FindByID(session *model.UserSession, id string) error {
	return r.db.Where("id = ?", id).First(session).Error
}

func (r *userSessionRepository) Delete(id string) error {
	return r.db.Delete(&model.UserSession{}, "id = ?", id).Error
}

func (r *userSessionRepository) RefreshTTL(id string, expiresAt time.Time) error {
	return r.db.Model(&model.UserSession{}).
		Where("id = ?", id).
		Update("expires_at", expiresAt).Error
}
