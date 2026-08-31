package model

import (
	"EquiliLearn/internal/entity"

	"github.com/google/uuid"
)

type UserRegister struct {
	Nama            string `json:"nama" validate:"required"`
	UserName        string `json:"username" validate:"required,min=3"`
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

type UserResponse struct {
	Id       uuid.UUID `json:"id"`
	Nama     string    `json:"nama" validate:"required"`
	UserName string    `json:"username" validate:"required,min=3"`
	Email    string    `json:"email" validate:"required,email"`
	Profil   string    `json:"profil"`
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
		Id:       User.Id,
		Nama:     User.Nama,
		UserName: User.UserName,
		Email:    User.Email,
	}
}

