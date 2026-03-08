package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/model"

	"github.com/labstack/echo/v4"
)

type mockSessionTTLRepository struct {
	findByIDFn   func(session *model.UserSession, id string) error
	deleteFn     func(id string) error
	refreshTTLFn func(id string, expiresAt time.Time) error
	createFn     func(session *model.UserSession) error
}

func (m *mockSessionTTLRepository) Create(session *model.UserSession) error {
	if m.createFn == nil {
		return nil
	}
	return m.createFn(session)
}

func (m *mockSessionTTLRepository) FindByID(session *model.UserSession, id string) error {
	if m.findByIDFn == nil {
		return nil
	}
	return m.findByIDFn(session, id)
}

func (m *mockSessionTTLRepository) Delete(id string) error {
	if m.deleteFn == nil {
		return nil
	}
	return m.deleteFn(id)
}

func (m *mockSessionTTLRepository) RefreshTTL(id string, expiresAt time.Time) error {
	if m.refreshTTLFn == nil {
		return nil
	}
	return m.refreshTTLFn(id, expiresAt)
}

func TestSessionTTLRefresh_DefaultConfig_SafeMethod(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			t.Fatal("RefreshTTL should not be called for safe methods")
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
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
		t.Fatal("next should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookies, got %d", len(rec.Result().Cookies()))
	}
}

func TestSessionTTLRefresh_ErrorResponse_NoRefresh(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			t.Fatal("RefreshTTL should not be called on error response")
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      24 * time.Hour,
		Now:      func() time.Time { return now },
	})

	next := func(c echo.Context) error {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad request"})
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookies, got %d", len(rec.Result().Cookies()))
	}
}

func TestSessionTTLRefresh_NoSessionCookie(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			t.Fatal("RefreshTTL should not be called when cookie missing")
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      24 * time.Hour,
		Now:      func() time.Time { return now },
	})

	next := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookies, got %d", len(rec.Result().Cookies()))
	}
}

func TestSessionTTLRefresh_EmptySessionCookie(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			t.Fatal("RefreshTTL should not be called when cookie empty")
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      24 * time.Hour,
		Now:      func() time.Time { return now },
	})

	next := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookies, got %d", len(rec.Result().Cookies()))
	}
}

func TestSessionTTLRefresh_Success_WithoutCSRFCookie(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	expectedExpires := now.Add(30 * time.Minute)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	refreshCalled := false
	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			refreshCalled = true
			if id != "session-123" {
				t.Fatalf("unexpected id: %s", id)
			}
			if !expiresAt.Equal(expectedExpires) {
				t.Fatalf("expected expiresAt %v, got %v", expectedExpires, expiresAt)
			}
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      30 * time.Minute,
		Now:      func() time.Time { return now },
	})

	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if !nextCalled {
		t.Fatal("next should be called")
	}
	if !refreshCalled {
		t.Fatal("RefreshTTL should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) >= 1 {
		ck := cookies[0]
		if ck.Name == "session_id" && ck.Value == "session-123" && ck.Path == "/" {
			if !ck.Expires.Equal(expectedExpires) {
				t.Fatalf("expected expires %v, got %v", expectedExpires, ck.Expires)
			}
		}
	}
}

func TestSessionTTLRefresh_Success_WithCSRFCookie(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	expectedExpires := now.Add(2 * time.Hour)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	refreshCalled := false
	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			refreshCalled = true
			if id != "session-123" {
				t.Fatalf("unexpected id: %s", id)
			}
			if !expiresAt.Equal(expectedExpires) {
				t.Fatalf("expected expiresAt %v, got %v", expectedExpires, expiresAt)
			}
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      2 * time.Hour,
		Now:      func() time.Time { return now },
	})

	next := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if !refreshCalled {
		t.Fatal("RefreshTTL should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) >= 2 {
		var sessionCookie, csrfCookie *http.Cookie
		for _, ck := range cookies {
			switch ck.Name {
			case "session_id":
				sessionCookie = ck
			case "csrf_token":
				csrfCookie = ck
			}
		}
		if sessionCookie != nil && sessionCookie.Value != "session-123" {
			t.Fatalf("unexpected session value: %s", sessionCookie.Value)
		}
		if csrfCookie != nil && csrfCookie.Value != "csrf-123" {
			t.Fatalf("unexpected csrf value: %s", csrfCookie.Value)
		}
	}
}

func TestSessionTTLRefresh_Success_EmptyCSRFCookieValue(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	repo := &mockSessionTTLRepository{
		refreshTTLFn: func(id string, expiresAt time.Time) error {
			return nil
		},
	}

	mw := SessionTTLRefresh(SessionTTLRefreshConfig{
		Sessions: repo,
		TTL:      time.Hour,
		Now:      func() time.Time { return now },
	})

	next := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	if err := mw(next)(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
