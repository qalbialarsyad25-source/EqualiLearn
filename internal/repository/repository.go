package repository

import (
	"gorm.io/gorm"
)

type Repository struct {
	UserRepository            IUserRepository
	TranscriptionRepository   ITranscriptionRepository
	DocumentSummaryRepository IDocumentSummaryRepository
	GroupChatRepository       IGroupChatRepository
}


func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		UserRepository:            NewUserRepository(db),
		TranscriptionRepository:   NewTranscriptionRepository(db),
		DocumentSummaryRepository: NewDocumentSummaryRepository(db),
		GroupChatRepository:       NewGroupChatRepository(db),
	}
}