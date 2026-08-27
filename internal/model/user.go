package model

import (
	"EquiliLearn/internal/entity"
	"github.com/google/uuid"
)


type UserRegister struct {
	Name            string `json:"name" validate:"required"`
	Email           string `json:"email" validate:"required"`
	Password        string `json:"password" validate:"required"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

type UserResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
}

type UserLogin struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func ToUserResponse(User entity.User) UserResponse {
	return UserResponse{
		Id:       User.Id,
		Name:     User.Name,
		Email:    User.Email,
	}
}