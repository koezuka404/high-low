package usecase

import (
	"testing"
)

func TestNormalizeEmailForRateLimit(t *testing.T) {
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
		got, err := NormalizeEmailForRateLimit(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeEmailForRateLimit(%q) want error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeEmailForRateLimit(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeEmailForRateLimit(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRateLimitIPKeyAndCost(t *testing.T) {
	def := 1.0
	t.Run("valid v4", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("  192.168.1.10  ", def)
		if s != "192.168.1.10" || c != def {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v4 with port", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("10.0.0.1:54321", def)
		if s != "10.0.0.1" || c != def {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("valid v6", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("2001:db8::1", def)
		if c != def || s == "" || s == rateLimitIPKeyUnknown {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v6 bracket", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("[::1]", def)
		if c != def || s != "::1" {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("v6 bracket port", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("[2001:db8::1]:443", def)
		if c != def || s == rateLimitIPKeyUnknown {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("empty unknown cost 0", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("  ", def)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
	t.Run("garbage unknown cost 0", func(t *testing.T) {
		s, c := RateLimitIPKeyAndCost("not-an-ip", def)
		if s != rateLimitIPKeyUnknown || c != 0 {
			t.Fatalf("got %q, %v", s, c)
		}
	})
}
