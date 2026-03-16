package usecase

import (
	"context"
	"time"

	"backend/model"
)

type IGameRepository interface {
	Create(game *model.Game) error
	GetGameByID(id uint) (*model.Game, error)
	GetGameByUserID(userID uint) (*model.Game, error)
	UpdateWithVersion(game *model.Game, expectedVer int64) error
}

type IGameRoundLogRepository interface {
	Create(log *model.GameRoundLog) error
	GetRoundLogsByGameID(gameID uint) ([]model.Round, error)
	GetRoundLogCountByGameID(gameID uint) (int64, error)
	DeleteByGameID(gameID uint) error
}

type IUserRepository interface {
	Create(user *model.User) error
	GetUserByEmail(user *model.User, email string) error
	GetUserByID(user *model.User, id uint) error
}

type IUserSessionRepository interface {
	Create(session *model.UserSession) error
	FindByID(session *model.UserSession, id string) error
	Delete(id string) error
	RefreshTTL(id string, expiresAt time.Time) error
}

type ConsumeOptions struct {
	Capacity   *int
	RefillRate *int
	TokenCost  *int
	TTLSec     *int
}

type RateLimiter interface {
	ConsumeToken(ctx context.Context, key string, opts *ConsumeOptions) (allowed bool, retryAfterSec int, err error)
}
