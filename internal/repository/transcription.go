package repository

import (
	"context"
	"errors"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITranscriptionRepository interface {
	CreateTranscription(ctx context.Context, transcription *entity.Transcription) error
	GetTranscriptionsByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Transcription, error)
	GetTranscriptionByID(ctx context.Context, id uuid.UUID) (*entity.Transcription, error)
	DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type TranscriptionRepository struct {
	db *gorm.DB
}

func NewTranscriptionRepository(db *gorm.DB) *TranscriptionRepository {
	return &TranscriptionRepository{db}
}

func (r *TranscriptionRepository) CreateTranscription(ctx context.Context, transcription *entity.Transcription) error {
	return r.db.WithContext(ctx).Create(transcription).Error
}

func (r *TranscriptionRepository) GetTranscriptionsByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Transcription, error) {
	var transcriptions []entity.Transcription
	query := r.db.WithContext(ctx)
	if userID == uuid.Nil {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Order("created_at DESC").
		Find(&transcriptions).Error
	if err != nil {
		return nil, err
	}
	return transcriptions, nil
}

func (r *TranscriptionRepository) GetTranscriptionByID(ctx context.Context, id uuid.UUID) (*entity.Transcription, error) {
	var transcription entity.Transcription
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&transcription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transcription, nil
}

func (r *TranscriptionRepository) DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if userID != uuid.Nil {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}
	result := query.Delete(&entity.Transcription{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
