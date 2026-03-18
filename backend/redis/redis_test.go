package redis

import (
	"os"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
)

func TestNewRedis_EmptyAddrReturnsNil(t *testing.T) {
	t.Setenv("REDIS_ADDR", "")
	c, err := NewRedis()
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatalf("expected nil client, got %T", c)
	}
}

func TestNewRedis_PingSuccess(t *testing.T) {
	mr := miniredis.RunT(t)
	t.Setenv("REDIS_ADDR", mr.Addr())

	c, err := NewRedis()
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected client, got nil")
	}
	_ = c.Close()
}

func TestNewRedis_PingError(t *testing.T) {
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")

	c, err := NewRedis()
	if err == nil {
		t.Fatal("expected error")
	}
	if c != nil {
		_ = c.Close()
	}
}

func TestNewRedis_LoadDotEnvNoop(t *testing.T) {
	origDir, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer os.Chdir(origDir)

	t.Setenv("REDIS_ADDR", "")
	c, err := NewRedis()
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatalf("expected nil client, got %T", c)
	}
}

