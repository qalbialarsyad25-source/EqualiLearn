package model

import (
	"EquiliLearn/internal/entity"

	"github.com/google/uuid"
)

type UserRegister struct {
	Name            string `json:"name" validate:"required"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

type UserResponse struct {
	ID      uuid.UUID `json:"id"`
	Name     string    `json:"name" validate:"required"`
	Email    string    `json:"email" validate:"required,email"`
}

type UserLogin struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func ToUserResponse(User entity.User) UserResponse {
	return UserResponse{
		ID:       User.ID,
		Name:     User.Name,
		Email:    User.Email,
	}
}

