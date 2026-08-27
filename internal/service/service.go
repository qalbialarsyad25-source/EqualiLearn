package usecase

import (
	websocket "EquilLearn/internal/controller/delivery"
	"EquilLearn/internal/repository"
	"EquilLearn/pkg/bcrypt"
	"EquilLearn/pkg/jwt"

	"golang.org/x/oauth2"
)

type Usecase struct {
	AuthUsecase         IAuthUsecase
}

func NewUsecase(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oauth *oauth2.Config, repository *repository.Repository, ws *websocket.WSManager) *Usecase {
	return &Usecase{
		AuthUsecase:         NewAuthUsecase(jwt, bcrypt, oauth, repository.UserRepository),
	}
}
