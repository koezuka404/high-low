package repository

import (
	"errors"
	"testing"
	"time"

	"backend/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSessionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&model.UserSession{}); err != nil {
		t.Fatalf("failed to migrate user_sessions table: %v", err)
	}

	return db
}

func TestNewUserSessionRepository(t *testing.T) {
	db := setupSessionTestDB(t)

	r := NewUserSessionRepository(db)
	if r == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestUserSessionRepository_Create_Success(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	expiresAt := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	session := &model.UserSession{
		ID:        "session-123",
		UserID:    1,
		ExpiresAt: expiresAt,
	}

	err := r.Create(session)
	if err != nil {
		t.Fatal(err)
	}

	var got model.UserSession
	if err := db.First(&got, "id = ?", "session-123").Error; err != nil {
		t.Fatal(err)
	}

	if got.ID != "session-123" {
		t.Fatalf("expected ID session-123, got %s", got.ID)
	}
	if got.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", got.UserID)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected ExpiresAt %v, got %v", expiresAt, got.ExpiresAt)
	}
}

func TestUserSessionRepository_Create_Error(t *testing.T) {
	db := setupSessionTestDB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewUserSessionRepository(db)

	session := &model.UserSession{
		ID:        "session-123",
		UserID:    1,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err = r.Create(session)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserSessionRepository_FindByID_Success(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	expiresAt := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	seed := model.UserSession{
		ID:        "session-123",
		UserID:    10,
		ExpiresAt: expiresAt,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	var got model.UserSession
	err := r.FindByID(&got, "session-123")
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != "session-123" {
		t.Fatalf("expected ID session-123, got %s", got.ID)
	}
	if got.UserID != 10 {
		t.Fatalf("expected UserID 10, got %d", got.UserID)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected ExpiresAt %v, got %v", expiresAt, got.ExpiresAt)
	}
}

func TestUserSessionRepository_FindByID_NotFound(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	var got model.UserSession
	err := r.FindByID(&got, "not-found")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUserSessionRepository_Delete_Success(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	seed := model.UserSession{
		ID:        "session-123",
		UserID:    10,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	err := r.Delete("session-123")
	if err != nil {
		t.Fatal(err)
	}

	var got model.UserSession
	err = db.First(&got, "id = ?", "session-123").Error
	if err == nil {
		t.Fatal("expected deleted record to be missing")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUserSessionRepository_Delete_NotFound_NoError(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	err := r.Delete("not-found")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestUserSessionRepository_Delete_Error(t *testing.T) {
	db := setupSessionTestDB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewUserSessionRepository(db)

	err = r.Delete("session-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserSessionRepository_RefreshTTL_Success(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	oldExpires := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	newExpires := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	seed := model.UserSession{
		ID:        "session-123",
		UserID:    10,
		ExpiresAt: oldExpires,
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	err := r.RefreshTTL("session-123", newExpires)
	if err != nil {
		t.Fatal(err)
	}

	var got model.UserSession
	if err := db.First(&got, "id = ?", "session-123").Error; err != nil {
		t.Fatal(err)
	}

	if !got.ExpiresAt.Equal(newExpires) {
		t.Fatalf("expected ExpiresAt %v, got %v", newExpires, got.ExpiresAt)
	}
}

func TestUserSessionRepository_RefreshTTL_NotFound_NoError(t *testing.T) {
	db := setupSessionTestDB(t)
	r := NewUserSessionRepository(db)

	err := r.RefreshTTL("not-found", time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestUserSessionRepository_RefreshTTL_Error(t *testing.T) {
	db := setupSessionTestDB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewUserSessionRepository(db)

	err = r.RefreshTTL("session-123", time.Now().Add(24*time.Hour))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
