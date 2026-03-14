package main

import (
	"time"

	"backend/controller"
	"backend/db"
	"backend/model"
	"backend/middleware"
	"backend/redis"
	"backend/repository"
	"backend/router"
	"backend/usecase"
	"log"
)

func main() {
	database, err := db.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	if err := database.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Game{}, &model.GameRoundLog{}); err != nil {
		log.Fatal(err)
	}

	redisClient, _ := redis.NewRedis()
	rateLimitRepo := repository.NewRateLimitRepository(redisClient)

	userRepo := repository.NewUserRepository(database)
	sessionRepo := repository.NewUserSessionRepository(database)
	gameRepo := repository.NewGameRepository(database)
	gameRoundLogRepo := repository.NewGameRoundLogRepository(database)

	userUsecase := usecase.NewUserUsecase(userRepo, sessionRepo)
	gameUsecase := usecase.NewGameUsecase(gameRepo, gameRoundLogRepo)

	userController := controller.NewUserController(userUsecase)
	gameController := controller.NewGameController(gameUsecase)

	rateLimitMW := middleware.NewRateLimitMiddleware(middleware.RateLimitConfig{
		RateLimitRepo: rateLimitRepo,
		Sessions:      sessionRepo,
		Now:           time.Now,
	})

	e := router.NewRouter(userController, sessionRepo, gameController, rateLimitMW)

	e.Logger.Fatal(e.Start(":8080"))
}
