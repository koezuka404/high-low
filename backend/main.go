package main

import (
	"backend/controller"
	"backend/db"
	"backend/model"
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

	userRepo := repository.NewUserRepository(database)
	sessionRepo := repository.NewUserSessionRepository(database)
	gameRepo := repository.NewGameRepository(database)
	gameRoundLogRepo := repository.NewGameRoundLogRepository(database)

	userUsecase := usecase.NewUserUsecase(userRepo, sessionRepo)
	gameUsecase := usecase.NewGameUsecase(gameRepo, gameRoundLogRepo)

	userController := controller.NewUserController(userUsecase)
	gameController := controller.NewGameController(gameUsecase)

	e := router.NewRouter(userController, sessionRepo, gameController)

	e.Logger.Fatal(e.Start(":8080"))
}
