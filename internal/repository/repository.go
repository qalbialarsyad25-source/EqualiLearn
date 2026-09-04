package repository

import (
	"gorm.io/gorm"
)

type Repository struct {
	UserRepository          IUserRepository
	TranscriptionRepository ITranscriptionRepository
}


func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		UserRepository:          NewUserRepository(db),
		TranscriptionRepository: NewTranscriptionRepository(db),
	}
}