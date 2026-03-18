package main

import (
	"os"
	"strconv"
	"time"

	"backend/controller"
	"backend/db"
	"backend/model"
	"backend/middleware"
	appredis "backend/redis"
	"backend/repository"
	"backend/router"
	"backend/usecase"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var runFn = func() error { return runWithDeps(defaultRunDeps()) }
var osExitImpl = os.Exit

func logErrDefault(err error) { log.Print(err) }
func exitDefault(code int)    { osExitImpl(code) }

var logErrFn = logErrDefault
var exitFn = exitDefault
var fatalFn = func(err error) {
	logErrFn(err)
	exitFn(1)
}

var getenvFn = os.Getenv
var newDBFn = db.NewDB
var autoMigrateFn = func(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Game{}, &model.GameRoundLog{})
}
var newRedisFn = appredis.NewRedis
var startAddrFn = func() string { return ":8080" }
var startFn = func(e *echo.Echo) error {
	return e.Start(startAddrFn())
}

func main() {
	if err := runFn(); err != nil {
		fatalFn(err)
	}
}

type runDeps struct {
	getenv func(string) string

	newDB      func() (*gorm.DB, error)
	autoMigrate func(db *gorm.DB) error
	newRedis   func() (redis.UniversalClient, error)
	start      func(e *echo.Echo) error

	onRateLimitParams func(usecase.RateLimitParams)
}

func defaultRunDeps() runDeps {
	return runDeps{
		getenv:      getenvFn,
		newDB:       newDBFn,
		autoMigrate: autoMigrateFn,
		newRedis:    newRedisFn,
		start:       startFn,
	}
}

func runWithDeps(deps runDeps) error {
	database, err := deps.newDB()
	if err != nil {
		return err
	}

	if err := deps.autoMigrate(database); err != nil {
		return err
	}

	redisClient, _ := deps.newRedis()

	rateLimitParams := usecase.RateLimitParams{
		Capacity:   20,
		RefillRate: 5,
		TokenCost:  1,
		TTLSec:     60,
	}
	if v := deps.getenv("RATE_LIMIT_CAPACITY"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.Capacity = n
		}
	}
	if v := deps.getenv("RATE_LIMIT_REFILL_RATE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.RefillRate = n
		}
	}
	if v := deps.getenv("RATE_LIMIT_TOKEN_COST"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n > 0 {
			rateLimitParams.TokenCost = n
		}
	}
	if v := deps.getenv("RATE_LIMIT_TTL_SEC"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			rateLimitParams.TTLSec = n
		}
	}
	if deps.onRateLimitParams != nil {
		deps.onRateLimitParams(rateLimitParams)
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
	return deps.start(e)
}
