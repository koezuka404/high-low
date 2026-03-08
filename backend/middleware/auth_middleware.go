package middleware

import (
	"net/http"
	"time"

	"backend/model"
	"backend/repository"

	"github.com/labstack/echo/v4"
)

const CtxUserIDKey = "user_id"

type AuthMiddleware struct {
	Sessions repository.IUserSessionRepository
	Now      func() time.Time
}

func NewAuthMiddleware(s repository.IUserSessionRepository) *AuthMiddleware {
	return &AuthMiddleware{
		Sessions: s,
		Now:      time.Now,
	}
}

func (m *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ck, err := c.Cookie("session_id")
		if err != nil || ck.Value == "" {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "unauthorized",
					"message": "session_id missing",
				},
			})
		}

		sess := model.UserSession{}
		if err := m.Sessions.FindByID(&sess, ck.Value); err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "unauthorized",
					"message": "invalid session",
				},
			})
		}

		now := m.Now()
		if !sess.ExpiresAt.After(now) {
			_ = m.Sessions.Delete(ck.Value)
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "unauthorized",
					"message": "session expired",
				},
			})
		}

		c.Set(CtxUserIDKey, sess.UserID)
		return next(c)
	}
}
