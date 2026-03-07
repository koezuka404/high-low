package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func CSRFMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		switch c.Request().Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return next(c)
		}

		csrfCookie, err := c.Cookie("csrf_token")
		if err != nil || csrfCookie.Value == "" {
			return c.JSON(http.StatusForbidden, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "forbidden",
					"message": "csrf token missing",
				},
			})
		}

		csrfHeader := c.Request().Header.Get("X-CSRF-Token")
		if csrfHeader == "" || csrfHeader != csrfCookie.Value {
			return c.JSON(http.StatusForbidden, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "forbidden",
					"message": "csrf token mismatch",
				},
			})
		}

		return next(c)
	}
}
