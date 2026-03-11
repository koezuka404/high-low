package router

import (
	"backend/controller"
	"backend/middleware"

	"github.com/labstack/echo/v4"
)

func RegisterGameRoutes(
	e *echo.Echo,
	gc controller.IGameController,
	rateLimitMW echo.MiddlewareFunc,
	authMW *middleware.AuthMiddleware,
	csrfMW echo.MiddlewareFunc,
	ttlMW echo.MiddlewareFunc,
) {
	g := e.Group("/api/game", rateLimitMW, authMW.RequireAuth)
	g.GET("/status", gc.Status)
	g.POST("/start", gc.Start, csrfMW, ttlMW)
	g.POST("/select", gc.Select, csrfMW, ttlMW)
	g.POST("/cheat", gc.Cheat, csrfMW, ttlMW)
	g.POST("/mode", gc.ChangeMode, csrfMW, ttlMW)
}
