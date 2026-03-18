package repository

import (
	"context"
	"errors"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestNewRateLimitRepository_NilClient_ReturnsNil(t *testing.T) {
	if got := NewRateLimitRepository(nil); got != nil {
		t.Fatalf("expected nil, got %T", got)
	}
}

func TestRateLimitRepository_ConsumeToken_AllowedAndDenied(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	ctx := context.Background()
	key := "ratelimit:test"

	allowed, retry, err := rl.ConsumeToken(ctx, key, 100, 1, 1, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed || retry != 0 {
		t.Fatalf("expected allowed, got allowed=%v retry=%d", allowed, retry)
	}

	allowed, retry, err = rl.ConsumeToken(ctx, key, 100, 1, 1, 1, 60)
	if err != nil {
		t.Fatal(err)
	}
	if allowed || retry < 1 {
		t.Fatalf("expected denied with retry>=1, got allowed=%v retry=%d", allowed, retry)
	}
}

func TestRateLimitRepository_ConsumeToken_ScriptError(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	mr.Close()
	_, _, err := rl.ConsumeToken(context.Background(), "ratelimit:x", 1, 1, 1, 1, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRateLimitRepository_ConsumeToken_UnexpectedResult(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	rl.script = redis.NewScript(`return 1`)
	_, _, err := rl.ConsumeToken(context.Background(), "ratelimit:x", 1, 1, 1, 1, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToInt64(t *testing.T) {
	if v, ok := toInt64(int64(3)); !ok || v != 3 {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
	if v, ok := toInt64(int(4)); !ok || v != 4 {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
	if _, ok := toInt64("x"); ok {
		t.Fatal("expected false")
	}
}

func TestRateLimitRepository_ConsumeToken_BadReturnShape(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	rl.script = redis.NewScript(`return {1}`)
	_, _, err := rl.ConsumeToken(context.Background(), "ratelimit:x", 1, 1, 1, 1, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRateLimitRepository_ConsumeToken_BadAllowedTypeStillWorks(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	rl.script = redis.NewScript(`return {"1", 0}`)
	allowed, _, err := rl.ConsumeToken(context.Background(), "ratelimit:x", 1, 1, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("expected allowed=false because toInt64 fails")
	}
}

func TestRateLimitRepository_ConsumeToken_ClientNilPanicsAvoidedByConstructor(t *testing.T) {
	if NewRateLimitRepository(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestRateLimitRepository_ConsumeToken_ScriptRunReturnsErrorWrapped(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := NewRateLimitRepository(client).(*rateLimitRepository)

	rl.script = redis.NewScript(`return redis.error_reply("boom")`)
	_, _, err := rl.ConsumeToken(context.Background(), "ratelimit:x", 1, 1, 1, 1, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, redis.Nil) && err.Error() == "" {
		t.Fatal("expected wrapped error message")
	}
}

