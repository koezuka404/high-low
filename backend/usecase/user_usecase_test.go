package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"backend/model"

	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	createFn         func(user *model.User) error
	getUserByEmailFn func(user *model.User, email string) error
	getUserByIDFn    func(user *model.User, id uint) error
}

func (m *mockUserRepository) Create(user *model.User) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(user)
}

func (m *mockUserRepository) GetUserByEmail(user *model.User, email string) error {
	if m.getUserByEmailFn == nil {
		return nil
	}
	return m.getUserByEmailFn(user, email)
}

func (m *mockUserRepository) GetUserByID(user *model.User, id uint) error {
	if m.getUserByIDFn == nil {
		return nil
	}
	return m.getUserByIDFn(user, id)
}

type mockUserSessionRepository struct {
	createFn    func(session *model.UserSession) error
	deleteFn    func(sessionID string) error
	findByIDFn  func(session *model.UserSession, id string) error
	refreshTTLFn func(id string, expiresAt time.Time) error
}

func (m *mockUserSessionRepository) Create(session *model.UserSession) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(session)
}

func (m *mockUserSessionRepository) Delete(sessionID string) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(sessionID)
}

func (m *mockUserSessionRepository) FindByID(session *model.UserSession, id string) error {
	if m.findByIDFn == nil {
		return nil
	}
	return m.findByIDFn(session, id)
}

func (m *mockUserSessionRepository) RefreshTTL(id string, expiresAt time.Time) error {
	if m.refreshTTLFn == nil {
		return nil
	}
	return m.refreshTTLFn(id, expiresAt)
}

var testRateLimitParams = RateLimitParams{
	Capacity:   20,
	RefillRate: 5,
	TokenCost:  1,
	TTLSec:     60,
}

func TestNewUserUsecase(t *testing.T) {
	uu := NewUserUsecase(&mockUserRepository{}, &mockUserSessionRepository{}, nil, testRateLimitParams)
	if uu == nil {
		t.Fatal("expected usecase, got nil")
	}
}

func TestUserUsecase_SignUp_Success(t *testing.T) {
	ur := &mockUserRepository{
		createFn: func(user *model.User) error {
			if user.Email != "test@example.com" {
				t.Fatalf("unexpected email: %s", user.Email)
			}
			if user.Password == "" {
				t.Fatal("expected hashed password")
			}
			if user.Password == "password123" {
				t.Fatal("password should be hashed")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")); err != nil {
				t.Fatalf("password was not hashed correctly: %v", err)
			}
			user.ID = 1
			return nil
		},
	}
	sr := &mockUserSessionRepository{}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	res, err := uu.SignUp(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "password123",
	}, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if res.ID != 1 {
		t.Fatalf("expected ID=1, got %d", res.ID)
	}
	if res.Email != "test@example.com" {
		t.Fatalf("unexpected email: %s", res.Email)
	}
}

func TestUserUsecase_SignUp_BcryptError(t *testing.T) {
	ur := &mockUserRepository{
		createFn: func(user *model.User) error {
			t.Fatal("Create should not be called when bcrypt fails")
			return nil
		},
	}
	sr := &mockUserSessionRepository{}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	_, err := uu.SignUp(context.Background(), model.User{
		Email:    "test@example.com",
		Password: strings.Repeat("a", 73),
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserUsecase_SignUp_CreateError(t *testing.T) {
	ur := &mockUserRepository{
		createFn: func(user *model.User) error {
			return errors.New("db error")
		},
	}
	sr := &mockUserSessionRepository{}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	_, err := uu.SignUp(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "password123",
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserUsecase_Login_GetUserByEmailError(t *testing.T) {
	ur := &mockUserRepository{
		getUserByEmailFn: func(user *model.User, email string) error {
			if email != "test@example.com" {
				t.Fatalf("unexpected email: %s", email)
			}
			return errors.New("user not found")
		},
	}
	sr := &mockUserSessionRepository{
		createFn: func(session *model.UserSession) error {
			t.Fatal("session Create should not be called")
			return nil
		},
	}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	_, err := uu.Login(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "password123",
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid credentials" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserUsecase_Login_ComparePasswordError(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	ur := &mockUserRepository{
		getUserByEmailFn: func(user *model.User, email string) error {
			user.ID = 10
			user.Email = email
			user.Password = string(hashed)
			return nil
		},
	}
	sr := &mockUserSessionRepository{
		createFn: func(session *model.UserSession) error {
			t.Fatal("session Create should not be called")
			return nil
		},
	}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	_, err = uu.Login(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "wrong-password",
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserUsecase_Login_CreateSessionError(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	ur := &mockUserRepository{
		getUserByEmailFn: func(user *model.User, email string) error {
			user.ID = 99
			user.Email = email
			user.Password = string(hashed)
			return nil
		},
	}
	sr := &mockUserSessionRepository{
		createFn: func(session *model.UserSession) error {
			if session.ID == "" {
				t.Fatal("expected session ID")
			}
			if session.UserID != 99 {
				t.Fatalf("expected UserID=99, got %d", session.UserID)
			}
			if session.ExpiresAt.IsZero() {
				t.Fatal("expected ExpiresAt")
			}
			return errors.New("session create error")
		},
	}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	_, err = uu.Login(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "password123",
	}, "127.0.0.1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "session create error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserUsecase_Login_Success(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	ur := &mockUserRepository{
		getUserByEmailFn: func(user *model.User, email string) error {
			if email != "test@example.com" {
				t.Fatalf("unexpected email: %s", email)
			}
			user.ID = 7
			user.Email = email
			user.Password = string(hashed)
			return nil
		},
	}

	sr := &mockUserSessionRepository{
		createFn: func(session *model.UserSession) error {
			if session.ID == "" {
				t.Fatal("expected session ID")
			}
			if session.UserID != 7 {
				t.Fatalf("expected UserID=7, got %d", session.UserID)
			}
			if session.ExpiresAt.IsZero() {
				t.Fatal("expected ExpiresAt")
			}
			return nil
		},
	}

	uu := NewUserUsecase(ur, sr, nil, testRateLimitParams)

	sessionID, err := uu.Login(context.Background(), model.User{
		Email:    "test@example.com",
		Password: "password123",
	}, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if sessionID == "" {
		t.Fatal("expected sessionID")
	}
}

func TestUserUsecase_Logout_Success(t *testing.T) {
	sr := &mockUserSessionRepository{
		deleteFn: func(sessionID string) error {
			if sessionID != "session-123" {
				t.Fatalf("unexpected sessionID: %s", sessionID)
			}
			return nil
		},
	}

	uu := NewUserUsecase(&mockUserRepository{}, sr, nil, testRateLimitParams)

	err := uu.Logout("session-123")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserUsecase_Logout_Error(t *testing.T) {
	sr := &mockUserSessionRepository{
		deleteFn: func(sessionID string) error {
			if sessionID != "session-123" {
				t.Fatalf("unexpected sessionID: %s", sessionID)
			}
			return errors.New("delete error")
		},
	}

	uu := NewUserUsecase(&mockUserRepository{}, sr, nil, testRateLimitParams)

	err := uu.Logout("session-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "delete error" {
		t.Fatalf("unexpected error: %v", err)
	}
}
