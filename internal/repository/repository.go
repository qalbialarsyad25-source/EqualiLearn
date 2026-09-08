package repository

import (
	"gorm.io/gorm"
)

type Repository struct {
	UserRepository            IUserRepository
	TranscriptionRepository   ITranscriptionRepository
	DocumentSummaryRepository IDocumentSummaryRepository
	TTSHistoryRepository      ITTSHistoryRepository
	GroupChatRepository       IGroupChatRepository
	HistoryRepository         IHistoryRepository
}


func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		UserRepository:            NewUserRepository(db),
		TranscriptionRepository:   NewTranscriptionRepository(db),
		DocumentSummaryRepository: NewDocumentSummaryRepository(db),
		TTSHistoryRepository:      NewTTSHistoryRepository(db),
		GroupChatRepository:       NewGroupChatRepository(db),
		HistoryRepository:         NewHistoryRepository(db),
	}
}