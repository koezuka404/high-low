package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/model"

	"github.com/labstack/echo/v4"
)

type mockUserController struct{}

func (m *mockUserController) SignUp(c echo.Context) error {
	return c.JSON(http.StatusCreated, model.ResponseUser{ID: 1, Email: "test@example.com"})
}

func (m *mockUserController) Login(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (m *mockUserController) Logout(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

type mockSessionRepo struct{}

func (m *mockSessionRepo) Create(session *model.UserSession) error { return nil }
func (m *mockSessionRepo) FindByID(session *model.UserSession, id string) error {
	session.ID = id
	session.UserID = 1
	session.ExpiresAt = time.Now().Add(time.Hour)
	return nil
}
func (m *mockSessionRepo) Delete(id string) error                          { return nil }
func (m *mockSessionRepo) RefreshTTL(id string, expiresAt time.Time) error { return nil }

type mockGameController struct{}

func (m *mockGameController) Start(c echo.Context) error       { return c.NoContent(http.StatusOK) }
func (m *mockGameController) Select(c echo.Context) error      { return c.NoContent(http.StatusOK) }
func (m *mockGameController) Cheat(c echo.Context) error      { return c.NoContent(http.StatusOK) }
func (m *mockGameController) ChangeMode(c echo.Context) error  { return c.NoContent(http.StatusOK) }
func (m *mockGameController) Status(c echo.Context) error     { return c.NoContent(http.StatusOK) }

func TestNewRouter(t *testing.T) {

	ctrl := &mockUserController{}
	repo := &mockSessionRepo{}
	gameCtrl := &mockGameController{}

	e := NewRouter(ctrl, repo, gameCtrl)

	if e == nil {
		t.Fatal("router should not be nil")
	}

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/signup"},
		{http.MethodPost, "/login"},
		{http.MethodPost, "/logout"},
	}

	for _, tt := range tests {

		req := httptest.NewRequest(tt.method, tt.path, nil)

		if tt.path == "/logout" {
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "csrf-123"})
			req.Header.Set("X-CSRF-Token", "csrf-123")
		}

		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code == 0 {
			t.Fatalf("route %s %s not registered", tt.method, tt.path)
		}
	}
}
