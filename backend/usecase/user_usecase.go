package usecase

import (
	"context"
	"fmt"
	"time"

	"backend/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RateLimitError struct {
	RetryAfterSec int
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded"
}

type IUserUsecase interface {
	SignUp(ctx context.Context, user model.User, clientIP string) (model.ResponseUser, error)
	Login(ctx context.Context, user model.User, clientIP string) (string, error)
	Logout(sessionID string) error
}

type userUsecase struct {
	ur   IUserRepository
	sr   IUserSessionRepository
	rl   RateLimiter
	rlp  RateLimitParams
}

func NewUserUsecase(
	ur IUserRepository,
	sr IUserSessionRepository,
	rl RateLimiter,
	rlp RateLimitParams,
) IUserUsecase {
	return &userUsecase{ur: ur, sr: sr, rl: rl, rlp: rlp}
}

func (uu *userUsecase) SignUp(ctx context.Context, user model.User, clientIP string) (model.ResponseUser, error) {
	emailNorm, err := NormalizeEmailForRateLimit(user.Email)
	if err != nil {
		return model.ResponseUser{}, err
	}
	user.Email = emailNorm

	if uu.rl != nil {
		ipSuffix, ipCost := RateLimitIPKeyAndCost(clientIP, uu.rlp.TokenCost)
		key := "ratelimit:signup:ip:" + ipSuffix
		now := float64(time.Now().Unix())
		allowed, retryAfterSec, err := uu.rl.ConsumeToken(ctx, key, now, uu.rlp.Capacity, uu.rlp.RefillRate, ipCost, uu.rlp.TTLSec)
		if err != nil {
			return model.ResponseUser{}, fmt.Errorf("rate limit check: %w", err)
		}
		if !allowed {
			return model.ResponseUser{}, &RateLimitError{RetryAfterSec: retryAfterSec}
		}
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return model.ResponseUser{}, err
	}

	newUser := model.User{
		Email:    user.Email,
		Password: string(hash),
	}

	if err := uu.ur.Create(&newUser); err != nil {
		return model.ResponseUser{}, err
	}

	return model.ResponseUser{
		ID:    newUser.ID,
		Email: newUser.Email,
	}, nil
}

func (uu *userUsecase) Login(ctx context.Context, user model.User, clientIP string) (string, error) {
	emailNorm, err := NormalizeEmailForRateLimit(user.Email)
	if err != nil {
		return "", err
	}
	user.Email = emailNorm

	if uu.rl != nil {
		now := float64(time.Now().Unix())
		ipSuffix, ipCost := RateLimitIPKeyAndCost(clientIP, uu.rlp.TokenCost)
		allowed, retryAfterSec, err := uu.rl.ConsumeToken(ctx, "ratelimit:login:ip:"+ipSuffix, now, uu.rlp.Capacity, uu.rlp.RefillRate, ipCost, uu.rlp.TTLSec)
		if err != nil {
			return "", fmt.Errorf("rate limit check: %w", err)
		}
		if !allowed {
			return "", &RateLimitError{RetryAfterSec: retryAfterSec}
		}
		allowed, retryAfterSec, err = uu.rl.ConsumeToken(ctx, "ratelimit:login:email:"+emailNorm, now, uu.rlp.Capacity, uu.rlp.RefillRate, uu.rlp.TokenCost, uu.rlp.TTLSec)
		if err != nil {
			return "", fmt.Errorf("rate limit check: %w", err)
		}
		if !allowed {
			return "", &RateLimitError{RetryAfterSec: retryAfterSec}
		}
	}

	storedUser := model.User{}

	if err := uu.ur.GetUserByEmail(&storedUser, user.Email); err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(storedUser.Password),
		[]byte(user.Password),
	); err != nil {
		return "", err
	}

	sessionID := uuid.NewString()

	session := model.UserSession{
		ID:        sessionID,
		UserID:    storedUser.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := uu.sr.Create(&session); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (uu *userUsecase) Logout(sessionID string) error {
	return uu.sr.Delete(sessionID)
}
