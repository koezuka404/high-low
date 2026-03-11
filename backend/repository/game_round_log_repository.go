package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type IGameRoundLogRepository interface {
	Create(log *model.GameRoundLog) error
	FindByGameID(gameID uint) ([]model.Round, error)
	CountByGameID(gameID uint) (int64, error)
	DeleteByGameID(gameID uint) error
}

type gameRoundLogRepository struct {
	db *gorm.DB
}

func NewGameRoundLogRepository(db *gorm.DB) IGameRoundLogRepository {
	return &gameRoundLogRepository{db: db}
}

func (r *gameRoundLogRepository) Create(log *model.GameRoundLog) error {
	return r.db.Create(log).Error
}

func (r *gameRoundLogRepository) FindByGameID(gameID uint) ([]model.Round, error) {
	var logs []model.GameRoundLog
	if err := r.db.Where("game_id = ?", gameID).Order("number ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	out := make([]model.Round, 0, len(logs))
	for _, l := range logs {
		out = append(out, model.Round{
			Number:           l.Number,
			PlayerCard:       l.PlayerCard,
			DealerCard:       l.DealerCard,
			Result:           l.Result,
			ConsecutiveDraws: l.ConsecutiveDraws,
			CheatUsed:        l.CheatUsed,
			PlayedAt:         l.PlayedAt,
		})
	}
	return out, nil
}

func (r *gameRoundLogRepository) CountByGameID(gameID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.GameRoundLog{}).Where("game_id = ?", gameID).Count(&n).Error
	return n, err
}

func (r *gameRoundLogRepository) DeleteByGameID(gameID uint) error {
	return r.db.Where("game_id = ?", gameID).Delete(&model.GameRoundLog{}).Error
}
