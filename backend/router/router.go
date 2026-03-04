package router

import (
	"backend/controller"

	"github.com/labstack/echo/v4"
)

func NewRouter(userController *controller.UserController) *echo.Echo {

	e := echo.New()

	api := e.Group("/api/auth")

	api.POST("/signup", userController.Signup)
	api.POST("/login", userController.Login)
	api.POST("/logout", userController.Logout)

	return e
}
