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
	ur IUserRepository
	sr IUserSessionRepository
	rl RateLimiter
}

func NewUserUsecase(
	ur IUserRepository,
	sr IUserSessionRepository,
	rl RateLimiter,
) IUserUsecase {
	return &userUsecase{ur: ur, sr: sr, rl: rl}
}

func (uu *userUsecase) SignUp(ctx context.Context, user model.User, clientIP string) (model.ResponseUser, error) {
	if uu.rl != nil {
		allowed, retryAfterSec, err := uu.rl.ConsumeToken(ctx, "signup:ip:"+clientIP, nil)
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
	if uu.rl != nil {
		allowed, retryAfterSec, err := uu.rl.ConsumeToken(ctx, "login:ip:"+clientIP, nil)
		if err != nil {
			return "", fmt.Errorf("rate limit check: %w", err)
		}
		if !allowed {
			return "", &RateLimitError{RetryAfterSec: retryAfterSec}
		}
		allowed, retryAfterSec, err = uu.rl.ConsumeToken(ctx, "login:email:"+user.Email, nil)
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
