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
	database := db.NewDB()

	if err := database.AutoMigrate(&model.User{}, &model.UserSession{}); err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(database)
	sessionRepo := repository.NewUserSessionRepository(database)

	userUsecase := usecase.NewUserUsecase(userRepo, sessionRepo)
	userController := controller.NewUserController(userUsecase)

	e := router.NewRouter(userController, sessionRepo)

	e.Logger.Fatal(e.Start(":8080"))
}
