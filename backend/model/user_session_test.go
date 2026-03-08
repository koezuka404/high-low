package model

import (
	"testing"
	"time"
)

func TestUserSession_TableName(t *testing.T) {
	var s UserSession

	name := s.TableName()

	if name != "user_sessions" {
		t.Fatalf("expected table name user_sessions, got %s", name)
	}
}

func TestUserSession_StructFields(t *testing.T) {

	now := time.Now()

	s := UserSession{
		ID:        "session-123",
		UserID:    10,
		ExpiresAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if s.ID != "session-123" {
		t.Fatalf("unexpected ID: %s", s.ID)
	}

	if s.UserID != 10 {
		t.Fatalf("unexpected UserID: %d", s.UserID)
	}

	if !s.ExpiresAt.Equal(now) {
		t.Fatalf("unexpected ExpiresAt")
	}

	if !s.CreatedAt.Equal(now) {
		t.Fatalf("unexpected CreatedAt")
	}

	if !s.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected UpdatedAt")
	}
}
