package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/model"
	"backend/usecase"

	"github.com/labstack/echo/v4"
)

type mockRateLimiter struct {
	allowed       bool
	retryAfterSec int
	err           error

	calls []string
}

func (m *mockRateLimiter) ConsumeToken(ctx context.Context, key string, now float64, capacity, refillRate, tokenCost float64, ttlSec int64) (bool, int, error) {
	m.calls = append(m.calls, key)
	if m.err != nil {
		return false, 0, m.err
	}
	return m.allowed, m.retryAfterSec, nil
}

type mockSessions struct {
	findByIDFn func(session *model.UserSession, id string) error
}

func (m *mockSessions) Create(session *model.UserSession) error { return nil }
func (m *mockSessions) Delete(id string) error                  { return nil }
func (m *mockSessions) RefreshTTL(id string, expiresAt time.Time) error {
	return nil
}
func (m *mockSessions) FindByID(session *model.UserSession, id string) error {
	if m.findByIDFn == nil {
		return nil
	}
	return m.findByIDFn(session, id)
}

func TestRateLimitMiddleware_SkipsStatusGET(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/game/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	rl := &mockRateLimiter{allowed: false}
	mw := NewRateLimitMiddleware(RateLimitConfig{
		RateLimitRepo: rl,
		Sessions:      &mockSessions{},
		Now:           time.Now,
		Params:        usecase.RateLimitParams{Capacity: 1, RefillRate: 1, TokenCost: 1, TTLSec: 60},
	})

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected next called")
	}
	if len(rl.calls) != 0 {
		t.Fatalf("expected no rate limit calls, got %v", rl.calls)
	}
}

func TestRateLimitMiddleware_SkipsWhenRepoNil(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/game/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := NewRateLimitMiddleware(RateLimitConfig{
		RateLimitRepo: nil,
		Sessions:      &mockSessions{},
		Now:           time.Now,
		Params:        usecase.RateLimitParams{Capacity: 1, RefillRate: 1, TokenCost: 1, TTLSec: 60},
	})

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}
	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected next called")
	}
}

func TestGetUserIDForRateLimit(t *testing.T) {
	e := echo.New()
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)

	t.Run("no cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if got := getUserIDForRateLimit(c, &mockSessions{}, func() time.Time { return now }); got != 0 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("empty cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: ""})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if got := getUserIDForRateLimit(c, &mockSessions{}, func() time.Time { return now }); got != 0 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("FindByID error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		s := &mockSessions{findByIDFn: func(session *model.UserSession, id string) error { return errors.New("nope") }}
		if got := getUserIDForRateLimit(c, s, func() time.Time { return now }); got != 0 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("expired session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "expired"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		s := &mockSessions{findByIDFn: func(session *model.UserSession, id string) error {
			session.ID = id
			session.UserID = 99
			session.ExpiresAt = now
			return nil
		}}
		if got := getUserIDForRateLimit(c, s, func() time.Time { return now }); got != 0 {
			t.Fatalf("got %d", got)
		}
	})

	t.Run("valid session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		s := &mockSessions{findByIDFn: func(session *model.UserSession, id string) error {
			session.ID = id
			session.UserID = 123
			session.ExpiresAt = now.Add(time.Hour)
			return nil
		}}
		if got := getUserIDForRateLimit(c, s, func() time.Time { return now }); got != 123 {
			t.Fatalf("got %d", got)
		}
	})
}

func TestRateLimitMiddleware_AllowsNext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/game/start", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	sessions := &mockSessions{findByIDFn: func(session *model.UserSession, id string) error {
		session.ID = id
		session.UserID = 7
		session.ExpiresAt = now.Add(time.Hour)
		return nil
	}}
	rl := &mockRateLimiter{allowed: true}
	mw := NewRateLimitMiddleware(RateLimitConfig{
		RateLimitRepo: rl,
		Sessions:      sessions,
		Now:           func() time.Time { return now },
		Params:        usecase.RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60},
	})

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}
	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected next called")
	}
	if len(rl.calls) != 1 || rl.calls[0] != "ratelimit:user:7" {
		t.Fatalf("unexpected calls: %v", rl.calls)
	}
}

func TestRateLimitMiddleware_Denied429(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/game/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	rl := &mockRateLimiter{allowed: false, retryAfterSec: 3}
	mw := NewRateLimitMiddleware(RateLimitConfig{
		RateLimitRepo: rl,
		Sessions:      &mockSessions{},
		Now:           time.Now,
		Params:        usecase.RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60},
	})

	next := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "3" {
		t.Fatalf("expected Retry-After=3, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestRateLimitMiddleware_Error500(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/game/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	rl := &mockRateLimiter{err: errors.New("redis down")}
	mw := NewRateLimitMiddleware(RateLimitConfig{
		RateLimitRepo: rl,
		Sessions:      &mockSessions{},
		Now:           time.Now,
		Params:        usecase.RateLimitParams{Capacity: 20, RefillRate: 5, TokenCost: 1, TTLSec: 60},
	})

	next := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

