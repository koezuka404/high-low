package usecase

import (
	"context"
	"errors"
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
	emailNorm, err := EnforceAuthRateLimit(ctx, uu.rl, uu.rlp, clientIP, user.Email, "ratelimit:signup:ip:", "ratelimit:signup:email:")
	if err != nil {
		return model.ResponseUser{}, err
	}
	user.Email = emailNorm

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
	emailNorm, err := EnforceAuthRateLimit(ctx, uu.rl, uu.rlp, clientIP, user.Email, "ratelimit:login:ip:", "ratelimit:login:email:")
	if err != nil {
		return "", err
	}
	user.Email = emailNorm

	storedUser := model.User{}

	if err := uu.ur.GetUserByEmail(&storedUser, user.Email); err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(storedUser.Password),
		[]byte(user.Password),
	); err != nil {
		return "", errors.New("invalid credentials")
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
