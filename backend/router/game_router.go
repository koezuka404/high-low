package router

import (
	"backend/controller"

	"github.com/labstack/echo/v4"
)

func RegisterGameRoutes(e *echo.Echo, gc controller.IGameController) {
	g := e.Group("/api/game")

	g.POST("/start", gc.StartGame)
	g.POST("/select", gc.SelectCard)
	g.GET("/state", gc.GetGameState)
}
