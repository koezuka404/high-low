package repository

import (
	"backend/model"
	"backend/usecase"

	"gorm.io/gorm"
)

type gameRoundLogRepository struct {
	db *gorm.DB
}

func NewGameRoundLogRepository(db *gorm.DB) usecase.IGameRoundLogRepository {
	return &gameRoundLogRepository{db: db}
}

func (r *gameRoundLogRepository) Create(log *model.GameRoundLog) error {
	return r.db.Create(log).Error
}

func (r *gameRoundLogRepository) GetRoundLogsByGameID(gameID uint) ([]model.Round, error) {
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

func (r *gameRoundLogRepository) GetRoundLogCountByGameID(gameID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.GameRoundLog{}).Where("game_id = ?", gameID).Count(&n).Error
	return n, err
}

func (r *gameRoundLogRepository) DeleteByGameID(gameID uint) error {
	return r.db.Where("game_id = ?", gameID).Delete(&model.GameRoundLog{}).Error
}
