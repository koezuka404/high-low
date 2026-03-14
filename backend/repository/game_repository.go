package repository

import (
	"backend/model"

	"gorm.io/gorm"
)

type IGameRepository interface {
	Create(game *model.Game) error
	GetGameByID(id uint) (*model.Game, error)
	GetGameByUserID(userID uint) (*model.Game, error)
	UpdateWithVersion(game *model.Game, expectedVer int64) error
}

type gameRepository struct {
	db *gorm.DB
}

func NewGameRepository(db *gorm.DB) IGameRepository {
	return &gameRepository{db: db}
}

func (r *gameRepository) Create(game *model.Game) error {
	return r.db.Create(game).Error
}

func (r *gameRepository) GetGameByID(id uint) (*model.Game, error) {
	var game model.Game
	err := r.db.First(&game, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &game, nil
}

func (r *gameRepository) GetGameByUserID(userID uint) (*model.Game, error) {
	var game model.Game
	err := r.db.Where("user_id = ?", userID).First(&game).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &game, nil
}

func (r *gameRepository) UpdateWithVersion(game *model.Game, expectedVer int64) error {
	result := r.db.Model(game).
		Where("id = ? AND ver = ?", game.ID, expectedVer).
		Updates(map[string]interface{}{
			"status":             game.Status,
			"mode":               game.Mode,
			"player_wins":        game.PlayerWins,
			"dealer_wins":        game.DealerWins,
			"consecutive_draws":   game.ConsecutiveDraws,
			"cheated":            game.Cheated,
			"cheat_reserved":     game.CheatReserved,
			"cheat_card":         game.CheatCard,
			"ver":                game.Ver,
			"player_used_cards":  game.PlayerUsedCards,
			"dealer_used_cards":  game.DealerUsedCards,
			"updated_at":         game.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
