package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// RateLimitMiddleware は仕様 11.5.13 に従い、Auth の前に実行する。
// 未実装時は通過するだけ。実装時は IRateLimitRepository を注入し、対象 API で Token Bucket を行う。
func RateLimitMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// GET /api/game/status は対象外（仕様 11.5.3）
		if c.Request().Method == http.MethodGet && strings.HasSuffix(c.Request().URL.Path, "/status") {
			return next(c)
		}
		// TODO: IRateLimitRepository を注入し、userID でトークン消費。429 時は next を呼ばずに返す。
		return next(c)
	}
}
