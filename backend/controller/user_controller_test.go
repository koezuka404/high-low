package controller

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/model"

	"github.com/labstack/echo/v4"
)

type mockUserUsecase struct {
	signUpFn func(user model.User) (model.ResponseUser, error)
	loginFn  func(user model.User) (string, error)
	logoutFn func(sessionID string) error
}

func (m *mockUserUsecase) SignUp(user model.User) (model.ResponseUser, error) {
	if m.signUpFn == nil {
		return model.ResponseUser{}, nil
	}
	return m.signUpFn(user)
}

func (m *mockUserUsecase) Login(user model.User) (string, error) {
	if m.loginFn == nil {
		return "", nil
	}
	return m.loginFn(user)
}

func (m *mockUserUsecase) Logout(sessionID string) error {
	if m.logoutFn == nil {
		return nil
	}
	return m.logoutFn(sessionID)
}

func TestNewUserController(t *testing.T) {
	uc := NewUserController(&mockUserUsecase{})
	if uc == nil {
		t.Fatal("expected controller, got nil")
	}
}

func TestUserController_SignUp_Success(t *testing.T) {
	e := echo.New()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		signUpFn: func(user model.User) (model.ResponseUser, error) {
			if user.Email != "test@example.com" {
				t.Fatalf("unexpected email: %s", user.Email)
			}
			return model.ResponseUser{
				ID:    1,
				Email: "test@example.com",
			}, nil
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.SignUp(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	var res struct {
		Success bool              `json:"success"`
		Data    model.ResponseUser `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatal("expected success true")
	}
	if res.Data.ID != 1 || res.Data.Email != "test@example.com" {
		t.Fatalf("unexpected response data: %+v", res.Data)
	}
}

func TestUserController_SignUp_BindError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(`{"email":`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	uc := &userController{uu: &mockUserUsecase{}}

	if err := uc.SignUp(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUserController_SignUp_BadRequestErrors(t *testing.T) {
	cases := []string{
		"email is required",
		"invalid email format",
		"password is required",
		"password must be at least 8 characters",
	}

	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			e := echo.New()
			body := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUU := &mockUserUsecase{
				signUpFn: func(user model.User) (model.ResponseUser, error) {
					return model.ResponseUser{}, errors.New(msg)
				},
			}

			uc := &userController{uu: mockUU}

			if err := uc.SignUp(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestUserController_SignUp_InternalServerError(t *testing.T) {
	e := echo.New()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		signUpFn: func(user model.User) (model.ResponseUser, error) {
			return model.ResponseUser{}, errors.New("db error")
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.SignUp(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUserController_Login_BindError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	uc := &userController{uu: &mockUserUsecase{}}

	if err := uc.Login(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUserController_Login_BadRequestErrors(t *testing.T) {
	cases := []string{
		"email is required",
		"invalid email format",
		"password is required",
		"password must be at least 8 characters",
	}

	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			e := echo.New()
			body := `{"email":"test@example.com","password":"password123"}`
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUU := &mockUserUsecase{
				loginFn: func(user model.User) (string, error) {
					return "", errors.New(msg)
				},
			}

			uc := &userController{uu: mockUU}

			if err := uc.Login(c); err != nil {
				t.Fatal(err)
			}

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestUserController_Login_InternalServerError(t *testing.T) {
	e := echo.New()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		loginFn: func(user model.User) (string, error) {
			return "", errors.New("login failed")
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Login(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUserController_Login_Success_NoDomain(t *testing.T) {
	t.Setenv("API_DOMAIN", "")

	e := echo.New()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		loginFn: func(user model.User) (string, error) {
			return "session-123", nil
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Login(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	var sessionCookie *http.Cookie
	var csrfCookie *http.Cookie

	for _, ck := range cookies {
		switch ck.Name {
		case "session_id":
			sessionCookie = ck
		case "csrf_token":
			csrfCookie = ck
		}
	}

	if sessionCookie == nil {
		t.Fatal("session cookie not found")
	}
	if csrfCookie == nil {
		t.Fatal("csrf cookie not found")
	}

	if sessionCookie.Value != "session-123" {
		t.Fatalf("unexpected session_id value: %s", sessionCookie.Value)
	}
	if sessionCookie.Path != "/" {
		t.Fatalf("unexpected session path: %s", sessionCookie.Path)
	}
	if sessionCookie.Domain != "" {
		t.Fatalf("expected empty domain, got %s", sessionCookie.Domain)
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected session HttpOnly=true")
	}
	if !sessionCookie.Secure {
		t.Fatal("expected session Secure=true")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session SameSite: %v", sessionCookie.SameSite)
	}

	if csrfCookie.Value == "" {
		t.Fatal("csrf token should not be empty")
	}
	if csrfCookie.Path != "/" {
		t.Fatalf("unexpected csrf path: %s", csrfCookie.Path)
	}
	if csrfCookie.Domain != "" {
		t.Fatalf("expected empty domain, got %s", csrfCookie.Domain)
	}
	if csrfCookie.HttpOnly {
		t.Fatal("expected csrf HttpOnly=false")
	}
	if !csrfCookie.Secure {
		t.Fatal("expected csrf Secure=true")
	}
	if csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected csrf SameSite: %v", csrfCookie.SameSite)
	}
}

func TestUserController_Login_Success_WithDomain(t *testing.T) {
	t.Setenv("API_DOMAIN", "example.com")

	e := echo.New()
	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		loginFn: func(user model.User) (string, error) {
			return "session-123", nil
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Login(c); err != nil {
		t.Fatal(err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	for _, ck := range cookies {
		if ck.Name == "session_id" || ck.Name == "csrf_token" {
			if ck.Domain != "example.com" {
				t.Fatalf("expected domain example.com, got %s", ck.Domain)
			}
		}
	}
}

func TestUserController_Logout_NoSessionCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	uc := &userController{uu: &mockUserUsecase{}}

	if err := uc.Logout(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUserController_Logout_EmptySessionCookie(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: ""})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	uc := &userController{uu: &mockUserUsecase{}}

	if err := uc.Logout(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUserController_Logout_InvalidSession(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad-session"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		logoutFn: func(sessionID string) error {
			if sessionID != "bad-session" {
				t.Fatalf("unexpected sessionID: %s", sessionID)
			}
			return errors.New("invalid session")
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Logout(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestUserController_Logout_Success_NoDomain(t *testing.T) {
	t.Setenv("API_DOMAIN", "")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		logoutFn: func(sessionID string) error {
			if sessionID != "session-123" {
				t.Fatalf("unexpected sessionID: %s", sessionID)
			}
			return nil
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Logout(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	var sessionCookie *http.Cookie
	var csrfCookie *http.Cookie

	for _, ck := range cookies {
		switch ck.Name {
		case "session_id":
			sessionCookie = ck
		case "csrf_token":
			csrfCookie = ck
		}
	}

	if sessionCookie == nil || csrfCookie == nil {
		t.Fatal("expected both session and csrf cookies")
	}

	if sessionCookie.Value != "" {
		t.Fatalf("expected empty session cookie, got %s", sessionCookie.Value)
	}
	if sessionCookie.Domain != "" {
		t.Fatalf("expected empty session domain, got %s", sessionCookie.Domain)
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected session HttpOnly=true")
	}
	if !sessionCookie.Secure {
		t.Fatal("expected session Secure=true")
	}
	if sessionCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session SameSite: %v", sessionCookie.SameSite)
	}

	if csrfCookie.Value != "" {
		t.Fatalf("expected empty csrf cookie, got %s", csrfCookie.Value)
	}
	if csrfCookie.Domain != "" {
		t.Fatalf("expected empty csrf domain, got %s", csrfCookie.Domain)
	}
	if csrfCookie.HttpOnly {
		t.Fatal("expected csrf HttpOnly=false")
	}
	if !csrfCookie.Secure {
		t.Fatal("expected csrf Secure=true")
	}
	if csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected csrf SameSite: %v", csrfCookie.SameSite)
	}
}

func TestUserController_Logout_Success_WithDomain(t *testing.T) {
	t.Setenv("API_DOMAIN", "example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-123"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mockUU := &mockUserUsecase{
		logoutFn: func(sessionID string) error {
			return nil
		},
	}

	uc := &userController{uu: mockUU}

	if err := uc.Logout(c); err != nil {
		t.Fatal(err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	for _, ck := range cookies {
		if ck.Name == "session_id" || ck.Name == "csrf_token" {
			if ck.Domain != "example.com" {
				t.Fatalf("expected domain example.com, got %s", ck.Domain)
			}
		}
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token := generateCSRFToken()
	if token == "" {
		t.Fatal("token should not be empty")
	}
	if len(token) != 64 {
		t.Fatalf("expected token length 64, got %d", len(token))
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("token should be hex string: %v", err)
	}
}

func TestGenerateCSRFToken_RandReadError(t *testing.T) {
	old := csrfRandRead
	defer func() { csrfRandRead = old }()
	csrfRandRead = func([]byte) (int, error) {
		return 0, errors.New("fake rand failure")
	}
	token := generateCSRFToken()
	if token == "" {
		t.Fatal("fallback token should not be empty")
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Fatalf("fallback token should be hex: %v", err)
	}
}

