package service

import (
	websocket "EquiliLearn/internal/controller/delivery"
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/bcrypt"
	"EquiliLearn/pkg/jwt"

	"golang.org/x/oauth2"
)

type Usecase struct {
	AuthUsecase IAuthUsecase
}

func NewUsecase(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oauth *oauth2.Config, repository *repository.Repository, ws *websocket.WSManager) *Usecase {
	return &Usecase{
		AuthUsecase: NewAuthUsecase(jwt, bcrypt, oauth, repository.UserRepository),
	}
}
