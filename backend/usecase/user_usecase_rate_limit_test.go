package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/model"
)

type mockRateLimiterForUser struct {
	allowedPerCall []bool
	retryAfterSec  int
	err            error
	calls          []string
}

func (m *mockRateLimiterForUser) ConsumeToken(ctx context.Context, key string, now float64, capacity, refillRate, tokenCost float64, ttlSec int64) (bool, int, error) {
	m.calls = append(m.calls, key)
	if m.err != nil {
		return false, 0, m.err
	}
	if len(m.allowedPerCall) > 0 {
		i := len(m.calls) - 1
		if i < len(m.allowedPerCall) {
			return m.allowedPerCall[i], m.retryAfterSec, nil
		}
	}
	return true, 0, nil
}

func TestUserUsecase_SignUp_RateLimited(t *testing.T) {
	ur := &mockUserRepository{}
	sr := &mockUserSessionRepository{}
	rl := &mockRateLimiterForUser{allowedPerCall: []bool{false}, retryAfterSec: 4}
	uu := NewUserUsecase(ur, sr, rl, testRateLimitParams)

	_, err := uu.SignUp(context.Background(), model.User{Email: "test@example.com", Password: "password123"}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("unexpected err type: %T", err)
	}
}

func TestUserUsecase_Login_RateLimiterErrorPropagatesAsInternal(t *testing.T) {
	hashed := []byte("$2a$10$0L7nE9bJwQ0f44k3sA1C5u9Qb8yYh2ZBf0rYI2KqFhZlYv7i4dQqG") // looks like bcrypt but may fail; not reached
	ur := &mockUserRepository{
		getUserByEmailFn: func(user *model.User, email string) error {
			user.ID = 1
			user.Email = email
			user.Password = string(hashed)
			return nil
		},
	}
	sr := &mockUserSessionRepository{
		createFn: func(session *model.UserSession) error {
			session.ExpiresAt = time.Now()
			return nil
		},
	}
	rl := &mockRateLimiterForUser{err: errors.New("redis down")}
	uu := NewUserUsecase(ur, sr, rl, testRateLimitParams)

	_, err := uu.Login(context.Background(), model.User{Email: "test@example.com", Password: "password123"}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RateLimitError); ok {
		t.Fatal("unexpected RateLimitError")
	}
}

