package router

import (
	"time"

	"backend/controller"
	"backend/middleware"
	"backend/repository"

	"github.com/labstack/echo/v4"
)

func NewRouter(
	userController controller.IUserController,
	userSessionRepo repository.IUserSessionRepository,
	gameController controller.IGameController,
) *echo.Echo {

	e := echo.New()

	authMW := middleware.NewAuthMiddleware(userSessionRepo)

	ttlMW := middleware.SessionTTLRefresh(middleware.SessionTTLRefreshConfig{
		Sessions: userSessionRepo,
		TTL:      24 * time.Hour,
	})

	e.POST("/signup", userController.SignUp)
	e.POST("/login", userController.Login)
	e.POST("/logout",
		userController.Logout,
		authMW.RequireAuth,
		middleware.CSRFMiddleware,
		ttlMW,
	)

	RegisterGameRoutes(e, gameController, middleware.RateLimitMiddleware, authMW, middleware.CSRFMiddleware, ttlMW)

	return e
}
