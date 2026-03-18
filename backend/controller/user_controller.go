package controller

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"backend/model"
	usecase "backend/usecase"

	"github.com/labstack/echo/v4"
)

type IUserController interface {
	SignUp(c echo.Context) error
	Login(c echo.Context) error
	Logout(c echo.Context) error
}

type userController struct {
	uu usecase.IUserUsecase
}

func NewUserController(uu usecase.IUserUsecase) IUserController {
	return &userController{uu}
}

func (uc *userController) SignUp(c echo.Context) error {
	user := model.User{}
	if err := c.Bind(&user); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid_input", err.Error())
	}
	clientIP := c.RealIP()
	if clientIP == "" {
		clientIP = c.Request().RemoteAddr
	}
	userRes, err := uc.uu.SignUp(c.Request().Context(), user, clientIP)
	if err != nil {
		var rateLimitErr *usecase.RateLimitError
		if errors.As(err, &rateLimitErr) {
			c.Response().Header().Set("Retry-After", strconv.Itoa(rateLimitErr.RetryAfterSec))
			return respondError(c, http.StatusTooManyRequests, "too_many_requests", "rate limit exceeded")
		}
		msg := err.Error()
		if msg == "email is required" || msg == "invalid email format" || msg == "password is required" || msg == "password must be at least 8 characters" {
			return respondError(c, http.StatusBadRequest, "invalid_input", msg)
		}
		return respondError(c, http.StatusInternalServerError, "internal_error", msg)
	}
	return respondSuccess(c, http.StatusCreated, userRes)
}

func (uc *userController) Login(c echo.Context) error {
	user := model.User{}
	if err := c.Bind(&user); err != nil {
		return respondError(c, http.StatusBadRequest, "invalid_input", err.Error())
	}
	clientIP := c.RealIP()
	if clientIP == "" {
		clientIP = c.Request().RemoteAddr
	}
	sessionID, err := uc.uu.Login(c.Request().Context(), user, clientIP)
	if err != nil {
		var rateLimitErr *usecase.RateLimitError
		if errors.As(err, &rateLimitErr) {
			c.Response().Header().Set("Retry-After", strconv.Itoa(rateLimitErr.RetryAfterSec))
			return respondError(c, http.StatusTooManyRequests, "too_many_requests", "rate limit exceeded")
		}
		msg := err.Error()
		if msg == "invalid credentials" {
			return respondError(c, http.StatusUnauthorized, "unauthorized", msg)
		}
		if msg == "email is required" || msg == "invalid email format" || msg == "password is required" || msg == "password must be at least 8 characters" {
			return respondError(c, http.StatusBadRequest, "invalid_input", msg)
		}
		return respondError(c, http.StatusInternalServerError, "internal_error", msg)
	}

	sessionCookie := new(http.Cookie)
	sessionCookie.Name = "session_id"
	sessionCookie.Value = sessionID
	sessionCookie.Expires = time.Now().Add(24 * time.Hour)
	sessionCookie.Path = "/"
	if d := os.Getenv("API_DOMAIN"); d != "" {
		sessionCookie.Domain = d
	}
	sessionCookie.HttpOnly = true
	sessionCookie.Secure = cookieSecure()
	sessionCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(sessionCookie)

	csrfToken := generateCSRFToken()

	csrfCookie := new(http.Cookie)
	csrfCookie.Name = "csrf_token"
	csrfCookie.Value = csrfToken
	csrfCookie.Expires = time.Now().Add(24 * time.Hour)
	csrfCookie.Path = "/"
	if d := os.Getenv("API_DOMAIN"); d != "" {
		csrfCookie.Domain = d
	}
	csrfCookie.HttpOnly = false
	csrfCookie.Secure = cookieSecure()
	csrfCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(csrfCookie)

	return respondSuccess(c, http.StatusOK, nil)
}

func (uc *userController) Logout(c echo.Context) error {

	ck, err := c.Cookie("session_id")
	if err != nil || ck.Value == "" {
		return respondError(c, http.StatusUnauthorized, "unauthorized", "session not found")
	}

	if err := uc.uu.Logout(ck.Value); err != nil {
		return respondError(c, http.StatusUnauthorized, "unauthorized", "invalid session")
	}

	sessionCookie := new(http.Cookie)
	sessionCookie.Name = "session_id"
	sessionCookie.Value = ""
	sessionCookie.Expires = time.Now()
	sessionCookie.Path = "/"
	if d := os.Getenv("API_DOMAIN"); d != "" {
		sessionCookie.Domain = d
	}
	sessionCookie.HttpOnly = true
	sessionCookie.Secure = cookieSecure()
	sessionCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(sessionCookie)

	csrfCookie := new(http.Cookie)
	csrfCookie.Name = "csrf_token"
	csrfCookie.Value = ""
	csrfCookie.Expires = time.Now()
	csrfCookie.Path = "/"
	if d := os.Getenv("API_DOMAIN"); d != "" {
		csrfCookie.Domain = d
	}
	csrfCookie.HttpOnly = false
	csrfCookie.Secure = cookieSecure()
	csrfCookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(csrfCookie)

	return respondSuccess(c, http.StatusOK, nil)
}

var csrfRandRead = rand.Read

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := csrfRandRead(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

func cookieSecure() bool {
	// Spec requires Secure cookies in production, but local http dev cannot use Secure cookies.
	// Set COOKIE_SECURE=false to allow local development.
	return os.Getenv("COOKIE_SECURE") != "false"
}
