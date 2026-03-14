package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/model"
	"backend/repository"

	"github.com/labstack/echo/v4"
)

type RateLimitConfig struct {
	RateLimitRepo repository.IRateLimitRepository
	Sessions      repository.IUserSessionRepository
	Now           func() time.Time
}

func NewRateLimitMiddleware(cfg RateLimitConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == http.MethodGet && strings.HasSuffix(c.Request().URL.Path, "/status") {
				return next(c)
			}
			if cfg.RateLimitRepo == nil {
				return next(c)
			}
			userID := getUserIDForRateLimit(c, cfg.Sessions, cfg.Now)
			ctx := c.Request().Context()
			if ctx == nil {
				ctx = context.Background()
			}
			allowed, retryAfterSec, err := cfg.RateLimitRepo.ConsumeToken(ctx, userID)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"success": false,
					"error": map[string]any{
						"code":    "internal_error",
						"message": "rate limit check failed",
					},
				})
			}
			if !allowed {
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"success": false,
					"error": map[string]any{
						"code":    "too_many_requests",
						"message": "rate limit exceeded",
					},
				})
			}
			return next(c)
		}
	}
}

func getUserIDForRateLimit(c echo.Context, sessions repository.IUserSessionRepository, now func() time.Time) uint {
	ck, err := c.Cookie("session_id")
	if err != nil || ck == nil || ck.Value == "" {
		return 0
	}
	var sess model.UserSession
	if err := sessions.FindByID(&sess, ck.Value); err != nil {
		return 0
	}
	if !sess.ExpiresAt.After(now()) {
		return 0
	}
	return sess.UserID
}
