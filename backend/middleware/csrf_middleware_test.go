package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			called := false
			next := func(c echo.Context) error {
				called = true
				return c.NoContent(http.StatusOK)
			}

			if err := CSRFMiddleware(next)(c); err != nil {
				t.Fatal(err)
			}

			if !called {
				t.Fatal("next should be called")
			}
			if rec.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}

func TestCSRFMiddleware_MissingCSRFCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := CSRFMiddleware(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFMiddleware_EmptyCSRFCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := CSRFMiddleware(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFMiddleware_MissingHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := CSRFMiddleware(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFMiddleware_MismatchHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-123"})
	req.Header.Set("X-CSRF-Token", "wrong-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := CSRFMiddleware(next)(c); err != nil {
		t.Fatal(err)
	}

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCSRFMiddleware_Success(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-123"})
	req.Header.Set("X-CSRF-Token", "token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	next := func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	}

	if err := CSRFMiddleware(next)(c); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("next should be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
