package usecase

import (
	"context"
	"testing"
)

type mockRateLimiter struct {
	calls []string

	allowed        bool
	retryAfterSec  int
	err            error
	allowedPerCall []bool
}

func (m *mockRateLimiter) ConsumeToken(ctx context.Context, key string, now float64, capacity, refillRate, tokenCost float64, ttlSec int64) (bool, int, error) {
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
	return m.allowed, m.retryAfterSec, nil
}

type secondCallErrRateLimiter struct {
	calls int
}

func (m *secondCallErrRateLimiter) ConsumeToken(ctx context.Context, key string, now float64, capacity, refillRate, tokenCost float64, ttlSec int64) (bool, int, error) {
	m.calls++
	if m.calls == 1 {
		return true, 0, nil
	}
	return false, 0, context.Canceled
}

func TestEmailKeyPartForAuthRateLimit(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"  Test@Example.COM  ", "test@example.com", false},
		{"a@b.co", "a@b.co", false},
		{"user+tag@sub.example.org", "user+tag@sub.example.org", false},
		{"", "", true},
		{"no-at", "", true},
		{"a@b@c.com", "", true},
		{"@example.com", "", true},
		{"user@", "", true},
	}
	for _, tt := range tests {
		got, err := EmailKeyPartForAuthRateLimit(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("EmailKeyPartForAuthRateLimit(%q) want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("EmailKeyPartForAuthRateLimit(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("EmailKeyPartForAuthRateLimit(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIPKeyPartAndCostForAuthRateLimit(t *testing.T) {
	def := 1.0
	t.Run("valid v4", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("  192.168.1.10  ", def)
		if s != "192.168.1.10" || c != def {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v4 with port", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("10.0.0.1:54321", def)
		if s != "10.0.0.1" || c != def {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("valid v6", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("2001:db8::1", def)
		if c != def || s == "" || s == rateLimitIPKeyUnknown {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v6 bracket", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("[::1]", def)
		if c != def || s != "::1" {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v6 bracket port", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("[2001:db8::1]:443", def)
		if c != def || s == rateLimitIPKeyUnknown {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("empty unknown cost 0", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("  ", def)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("garbage unknown cost 0", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("not-an-ip", def)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("defaultTokenCost negative becomes 0", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("192.168.1.1", -5)
		if s != "192.168.1.1" || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("bracket parse fails (bad ip)", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("[bad]", 1)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("bracket missing closing", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("[::1", 1)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("empty brackets", func(t *testing.T) {
		s, c := IPKeyPartAndCostForAuthRateLimit("[]", 1)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
}

func TestEnforceAuthRateLimit_OrderAndKeys(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	rl := &mockRateLimiter{allowed: true}

	emailNorm, err := EnforceAuthRateLimit(
		context.Background(),
		rl,
		rlp,
		"10.0.0.1:1234",
		" Test@Example.COM ",
		"ratelimit:login:ip:",
		"ratelimit:login:email:",
	)
	if err != nil {
		t.Fatal(err)
	}
	if emailNorm != "test@example.com" {
		t.Fatalf("emailNorm = %q", emailNorm)
	}
	if len(rl.calls) != 2 {
		t.Fatalf("calls=%v", rl.calls)
	}
	if rl.calls[0] != "ratelimit:login:ip:10.0.0.1" {
		t.Fatalf("ip call key=%q", rl.calls[0])
	}
	if rl.calls[1] != "ratelimit:login:email:test@example.com" {
		t.Fatalf("email call key=%q", rl.calls[1])
	}
}

func TestEnforceAuthRateLimit_BlocksOnIP(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	rl := &mockRateLimiter{allowedPerCall: []bool{false}, retryAfterSec: 3}

	_, err := EnforceAuthRateLimit(
		context.Background(),
		rl,
		rlp,
		"10.0.0.1",
		"test@example.com",
		"ratelimit:login:ip:",
		"ratelimit:login:email:",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("unexpected err type: %T", err)
	}
	if len(rl.calls) != 1 {
		t.Fatalf("calls=%v", rl.calls)
	}
}

func TestEnforceAuthRateLimit_BlocksOnEmail(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	rl := &mockRateLimiter{allowedPerCall: []bool{true, false}, retryAfterSec: 2}

	_, err := EnforceAuthRateLimit(
		context.Background(),
		rl,
		rlp,
		"10.0.0.1",
		"test@example.com",
		"ratelimit:login:ip:",
		"ratelimit:login:email:",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("unexpected err type: %T", err)
	}
	if len(rl.calls) != 2 {
		t.Fatalf("calls=%v", rl.calls)
	}
}

func TestEnforceAuthRateLimit_InvalidEmail(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	rl := &mockRateLimiter{allowed: true}
	if _, err := EnforceAuthRateLimit(context.Background(), rl, rlp, "10.0.0.1", "bad", "ratelimit:x:", "ratelimit:y:"); err == nil {
		t.Fatal("expected error")
	}
	if len(rl.calls) != 0 {
		t.Fatalf("expected no calls, got %v", rl.calls)
	}
}

func TestEnforceAuthRateLimit_RateLimiterNil(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	emailNorm, err := EnforceAuthRateLimit(context.Background(), nil, rlp, "10.0.0.1", "Test@Example.com", "ratelimit:x:", "ratelimit:y:")
	if err != nil {
		t.Fatal(err)
	}
	if emailNorm != "test@example.com" {
		t.Fatalf("emailNorm=%q", emailNorm)
	}
}

func TestEnforceAuthRateLimit_RateLimiterErrorOnIP(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	rl := &mockRateLimiter{err: context.Canceled}
	if _, err := EnforceAuthRateLimit(context.Background(), rl, rlp, "10.0.0.1", "test@example.com", "ratelimit:x:", "ratelimit:y:"); err == nil {
		t.Fatal("expected error")
	}
	if len(rl.calls) != 1 {
		t.Fatalf("calls=%v", rl.calls)
	}
}

func TestEnforceAuthRateLimit_RateLimiterErrorOnEmail(t *testing.T) {
	rlp := RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60}
	var rl secondCallErrRateLimiter
	_, err := EnforceAuthRateLimit(context.Background(), &rl, rlp, "10.0.0.1", "test@example.com", "ratelimit:x:", "ratelimit:y:")
	if err == nil {
		t.Fatal("expected error")
	}
	if rl.calls != 2 {
		t.Fatalf("calls=%d", rl.calls)
	}
}

