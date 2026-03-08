package repository

import (
	"errors"
	"testing"

	"backend/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	return db
}

func TestNewUserRepository(t *testing.T) {
	db := setupTestDB(t)

	r := NewUserRepository(db)
	if r == nil {
		t.Fatal("expected repository, got nil")
	}
}

func TestUserRepository_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)

	user := &model.User{
		Email:    "test@example.com",
		Password: "hashed-password",
	}

	err := r.Create(user)
	if err != nil {
		t.Fatal(err)
	}

	if user.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	var got model.User
	if err := db.First(&got, user.ID).Error; err != nil {
		t.Fatal(err)
	}

	if got.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", got.Email)
	}
	if got.Password != "hashed-password" {
		t.Fatalf("expected password hashed-password, got %s", got.Password)
	}
}

func TestUserRepository_Create_Error(t *testing.T) {
	db := setupTestDB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := NewUserRepository(db)

	user := &model.User{
		Email:    "test@example.com",
		Password: "hashed-password",
	}

	err = r.Create(user)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserRepository_GetUserByEmail_Success(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)

	seed := model.User{
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	var got model.User
	err := r.GetUserByEmail(&got, "test@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != seed.ID {
		t.Fatalf("expected ID %d, got %d", seed.ID, got.ID)
	}
	if got.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", got.Email)
	}
}

func TestUserRepository_GetUserByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)

	var got model.User
	err := r.GetUserByEmail(&got, "notfound@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUserRepository_GetUserByID_Success(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)

	seed := model.User{
		Email:    "test@example.com",
		Password: "hashed-password",
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatal(err)
	}

	var got model.User
	err := r.GetUserByID(&got, seed.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != seed.ID {
		t.Fatalf("expected ID %d, got %d", seed.ID, got.ID)
	}
	if got.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", got.Email)
	}
}

func TestUserRepository_GetUserByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	r := NewUserRepository(db)

	var got model.User
	err := r.GetUserByID(&got, 999999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}
