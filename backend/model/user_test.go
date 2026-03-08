package model

import (
	"testing"
	"time"
)

func TestUser_TableName(t *testing.T) {
	var u User

	if u.TableName() != "users" {
		t.Fatalf("expected table name users, got %s", u.TableName())
	}
}

func TestUser_StructFields(t *testing.T) {

	now := time.Now()

	u := User{
		ID:        1,
		Email:     "test@example.com",
		Password:  "hashed-password",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if u.ID != 1 {
		t.Fatalf("unexpected ID: %d", u.ID)
	}

	if u.Email != "test@example.com" {
		t.Fatalf("unexpected Email: %s", u.Email)
	}

	if u.Password != "hashed-password" {
		t.Fatalf("unexpected Password: %s", u.Password)
	}

	if !u.IsActive {
		t.Fatalf("expected IsActive true")
	}

	if !u.CreatedAt.Equal(now) {
		t.Fatalf("unexpected CreatedAt")
	}

	if !u.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected UpdatedAt")
	}
}

func TestRequestBodyUser(t *testing.T) {

	req := RequestBodyUser{
		Email:    "test@example.com",
		Password: "password123",
	}

	if req.Email != "test@example.com" {
		t.Fatalf("unexpected Email: %s", req.Email)
	}

	if req.Password != "password123" {
		t.Fatalf("unexpected Password")
	}
}

func TestResponseUser(t *testing.T) {

	res := ResponseUser{
		ID:    5,
		Email: "test@example.com",
	}

	if res.ID != 5 {
		t.Fatalf("unexpected ID: %d", res.ID)
	}

	if res.Email != "test@example.com" {
		t.Fatalf("unexpected Email: %s", res.Email)
	}
}
