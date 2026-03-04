package usecase

import (
	"time"

	"backend/model"
	"backend/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type IUserUsecase interface {
	SignUp(user model.User) (model.ResponseUser, error)
	Login(user model.User) (string, error)
	Logout(sessionID string) error
}

type userUsecase struct {
	ur repository.IUserRepository
	sr repository.IUserSessionRepository
}

func NewUserUsecase(
	ur repository.IUserRepository,
	sr repository.IUserSessionRepository,
) IUserUsecase {
	return &userUsecase{ur, sr}
}

func (uu *userUsecase) SignUp(user model.User) (model.ResponseUser, error) {

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return model.ResponseUser{}, err
	}

	newUser := model.User{
		Email:    user.Email,
		Password: string(hash),
	}

	if err := uu.ur.Create(&newUser); err != nil {
		return model.ResponseUser{}, err
	}

	return model.ResponseUser{
		ID:    newUser.ID,
		Email: newUser.Email,
	}, nil
}

func (uu *userUsecase) Login(user model.User) (string, error) {

	storedUser := model.User{}

	if err := uu.ur.GetUserByEmail(&storedUser, user.Email); err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(storedUser.Password),
		[]byte(user.Password),
	); err != nil {
		return "", err
	}

	sessionID := uuid.NewString()

	session := model.UserSession{
		ID:        sessionID,
		UserID:    storedUser.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := uu.sr.Create(&session); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (uu *userUsecase) Logout(sessionID string) error {
	return uu.sr.Delete(sessionID)
}
