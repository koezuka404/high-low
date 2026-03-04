package controller

import (
	"net/http"
	"time"

	"backend/model"
	"backend/usecase"

	"github.com/labstack/echo/v4"
)

type UserController struct {
	uu usecase.IUserUsecase
}

func NewUserController(uu usecase.IUserUsecase) *UserController {
	return &UserController{uu}
}

func (uc *UserController) Signup(c echo.Context) error {
	var body model.RequestBodyUser
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	user := model.User{
		Email:    body.Email,
		Password: body.Password,
	}

	resUser, err := uc.uu.SignUp(user)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, resUser)
}

func (uc *UserController) Login(c echo.Context) error {

	var body model.RequestBodyUser

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	user := model.User{
		Email:    body.Email,
		Password: body.Password,
	}

	sessionID, err := uc.uu.Login(user)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, err)
	}

	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}

	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, "login success")
}

func (uc *UserController) Logout(c echo.Context) error {

	cookie, err := c.Cookie("session_id")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, err)
	}

	if err := uc.uu.Logout(cookie.Value); err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	clearCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	c.SetCookie(clearCookie)

	return c.JSON(http.StatusOK, "logout success")
}
