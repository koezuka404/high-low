package db

import (
	"os"
	"strings"
	"testing"
)

func TestNewDB_EmptyDSN(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Unsetenv("DB_DSN")

	_, err := NewDB()
	if err == nil {
		t.Fatal("expected error when DB_DSN is empty")
	}
	if err.Error() != "DB_DSN is empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewDB_Success_SQLite(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Setenv("DB_DSN", "file::memory:?cache=shared")

	db, err := NewDB()
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if db == nil {
		t.Fatal("expected db instance, got nil")
	}
}

func TestNewDB_PostgresBranch(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Setenv("DB_DSN", "host=invalid.invalid user=x password=x dbname=x port=5432 sslmode=disable")

	_, err := NewDB()
	if err == nil {
		t.Fatal("expected error for invalid postgres DSN")
	}
}

func TestNewDB_Success_Postgres(t *testing.T) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" || dsn == ":memory:" || strings.HasPrefix(dsn, "file:") {
		t.Skip("DB_DSN not set or not postgres, skipping")
	}

	db, err := NewDB()
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}
	if db == nil {
		t.Fatal("expected db instance, got nil")
	}
}
