package model

import (
	"EquiliLearn/internal/entity"
	"github.com/google/uuid"
)


type UserRegister struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Password string    `json:"password"`
}

type UserLogin struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type EditUser struct {
	Name string `json:"name"`
}

func ToUserResponse(User entity.User) UserResponse {
	return UserResponse{
		Id:       User.Id,
		Name:     User.Name,
		Email:    User.Email,
		Password: User.Password,
	}
}

func (p *EditUser) ToMap() map[string]any {
	Update := map[string]any{}

	if p.Name != "" {
		Update["name"] = p.Name
	}

	return Update
}