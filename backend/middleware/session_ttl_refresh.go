package middleware

import (
	"net/http"
	"time"

	"backend/usecase"

	"github.com/labstack/echo/v4"
)

type SessionTTLRefreshConfig struct {
	Sessions usecase.IUserSessionRepository
	TTL      time.Duration
	Now      func() time.Time
}

func SessionTTLRefresh(cfg SessionTTLRefreshConfig) echo.MiddlewareFunc {
	if cfg.TTL == 0 {
		cfg.TTL = 24 * time.Hour
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			switch c.Request().Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return err
			}

			if c.Response().Status >= 400 {
				return err
			}

			ck, ckErr := c.Cookie("session_id")
			if ckErr != nil || ck.Value == "" {
				return err
			}

			expiresAt := cfg.Now().Add(cfg.TTL)

			_ = cfg.Sessions.RefreshTTL(ck.Value, expiresAt)

			c.SetCookie(&http.Cookie{
				Name:     "session_id",
				Value:    ck.Value,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
				Expires:  expiresAt,
			})

			if csrfCk, e := c.Cookie("csrf_token"); e == nil && csrfCk.Value != "" {
				c.SetCookie(&http.Cookie{
					Name:     "csrf_token",
					Value:    csrfCk.Value,
					Path:     "/",
					HttpOnly: false,
					Secure:   true,
					SameSite: http.SameSiteLaxMode,
					Expires:  expiresAt,
				})
			}

			return err
		}
	}
}
