package main

import (
	"os"
	"strconv"
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
	rateLimitParams := usecase.RateLimitParams{
		Capacity:   20,
		RefillRate: 5,
		TokenCost:  1,
		TTLSec:     60,
	}
	if v := os.Getenv("RATE_LIMIT_CAPACITY"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.Capacity = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_REFILL_RATE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.RefillRate = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_TOKEN_COST"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.TokenCost = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_TTL_SEC"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			rateLimitParams.TTLSec = n
		}
	}
	rateLimitRepo := repository.NewRateLimitRepository(redisClient)

	userRepo := repository.NewUserRepository(database)
	sessionRepo := repository.NewUserSessionRepository(database)
	gameRepo := repository.NewGameRepository(database)
	gameRoundLogRepo := repository.NewGameRoundLogRepository(database)

	userUsecase := usecase.NewUserUsecase(userRepo, sessionRepo, rateLimitRepo, rateLimitParams)
	gameUsecase := usecase.NewGameUsecase(gameRepo, gameRoundLogRepo)

	userController := controller.NewUserController(userUsecase)
	gameController := controller.NewGameController(gameUsecase)

	rateLimitMW := middleware.NewRateLimitMiddleware(middleware.RateLimitConfig{
		RateLimitRepo: rateLimitRepo,
		Sessions:      sessionRepo,
		Now:           time.Now,
		Params:        rateLimitParams,
	})

	e := router.NewRouter(userController, sessionRepo, gameController, rateLimitMW)

	e.Logger.Fatal(e.Start(":8080"))
}
