package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/model"

	"github.com/labstack/echo/v4"
)

type mockUserSessionRepository struct {
	findByIDFn   func(session *model.UserSession, id string) error
	deleteFn    func(id string) error
	refreshTTLFn func(id string, expiresAt time.Time) error
}

func (m *mockUserSessionRepository) Create(session *model.UserSession) error {
	return nil
}

func (m *mockUserSessionRepository) FindByID(session *model.UserSession, id string) error {
	if m.findByIDFn == nil {
		return nil
	}
	return m.findByIDFn(session, id)
}

func (m *mockUserSessionRepository) Delete(id string) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(id)
}

func (m *mockUserSessionRepository) RefreshTTL(id string, expiresAt time.Time) error {
	if m.refreshTTLFn == nil {
		return nil
	}
	return m.refreshTTLFn(id, expiresAt)
}

func TestNewAuthMiddleware(t *testing.T) {
	m := NewAuthMiddleware(&mockUserSessionRepository{})
	if m == nil {
		t.Fatal("expected middleware, got nil")
	}
	if m.Now == nil {
		t.Fatal("expected Now func")
	}
	if m.Sessions == nil {
		t.Fatal("expected Sessions repo")
	}
}

func TestRequireAuth_NoCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockUserSessionRepository{
		findByIDFn: func(session *model.UserSession, id string) error {
			t.Fatal("FindByID should not be called")
			return nil
		},
	}

	m := NewAuthMiddleware(repo)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := m.RequireAuth(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuth_EmptyCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockUserSessionRepository{
		findByIDFn: func(session *model.UserSession, id string) error {
			t.Fatal("FindByID should not be called")
			return nil
		},
	}

	m := NewAuthMiddleware(repo)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := m.RequireAuth(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuth_FindByIDError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockUserSessionRepository{
		findByIDFn: func(session *model.UserSession, id string) error {
			if id != "bad-session" {
				t.Fatalf("unexpected id: %s", id)
			}
			return errors.New("not found")
		},
		deleteFn: func(id string) error {
			t.Fatal("Delete should not be called")
			return nil
		},
	}

	m := NewAuthMiddleware(repo)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := m.RequireAuth(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuth_ExpiredSession(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "expired-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	deleteCalled := false

	repo := &mockUserSessionRepository{
		findByIDFn: func(session *model.UserSession, id string) error {
			if id != "expired-session" {
				t.Fatalf("unexpected id: %s", id)
			}
			session.ID = id
			session.UserID = 10
			session.ExpiresAt = now
			return nil
		},
		deleteFn: func(id string) error {
			deleteCalled = true
			if id != "expired-session" {
				t.Fatalf("unexpected delete id: %s", id)
			}
			return nil
		},
	}

	m := NewAuthMiddleware(repo)
	m.Now = func() time.Time { return now }

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := m.RequireAuth(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if !deleteCalled {
		t.Fatal("Delete should be called for expired session")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRequireAuth_Success(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockUserSessionRepository{
		findByIDFn: func(session *model.UserSession, id string) error {
			if id != "valid-session" {
				t.Fatalf("unexpected id: %s", id)
			}
			session.ID = id
			session.UserID = 123
			session.ExpiresAt = now.Add(1 * time.Hour)
			return nil
		},
		deleteFn: func(id string) error {
			t.Fatal("Delete should not be called")
			return nil
		},
	}

	m := NewAuthMiddleware(repo)
	m.Now = func() time.Time { return now }

	called := false
	next := func(c echo.Context) error {
		called = true

		v := c.Get(CtxUserIDKey)
		if v == nil {
			t.Fatal("expected user_id in context")
		}

		userID, ok := v.(uint)
		if !ok {
			t.Fatalf("expected uint user_id, got %T", v)
		}
		if userID != 123 {
			t.Fatalf("expected user_id=123, got %d", userID)
		}

		return c.NoContent(http.StatusOK)
	}

	if err := m.RequireAuth(next)(c); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("next should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
