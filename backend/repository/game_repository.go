package repository

import "backend/model"

type IGameRepository interface {
	Create(game *model.Game) error
	Save(game *model.Game) error
	FindByUserID(userID uint) (*model.Game, error)
}
