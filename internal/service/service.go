package service

import (
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/bcrypt"
	"EquiliLearn/pkg/jwt"

	"golang.org/x/oauth2"
)

type Service struct {
	AuthService         IAuthService
}

func NewService(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oauth *oauth2.Config, repository *repository.Repository) *Service {
	return &Service{
		AuthService:         NewAuthService(jwt, bcrypt, oauth, repository.UserRepository),
	}
}
