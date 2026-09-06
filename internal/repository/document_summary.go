package repository

import (
	"context"
	"errors"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IDocumentSummaryRepository interface {
	CreateDocumentSummary(ctx context.Context, summary *entity.DocumentSummary) error
	GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.DocumentSummary, int64, error)
	GetSummaryByID(ctx context.Context, id uuid.UUID) (*entity.DocumentSummary, error)
	DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type DocumentSummaryRepository struct {
	db *gorm.DB
}

func NewDocumentSummaryRepository(db *gorm.DB) *DocumentSummaryRepository {
	return &DocumentSummaryRepository{db: db}
}

func (r *DocumentSummaryRepository) CreateDocumentSummary(ctx context.Context, summary *entity.DocumentSummary) error {
	return r.db.WithContext(ctx).Create(summary).Error
}

func (r *DocumentSummaryRepository) GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.DocumentSummary, int64, error) {
	var summaries []entity.DocumentSummary
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.DocumentSummary{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Order("created_at DESC").
		Find(&summaries).Error
	if err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *DocumentSummaryRepository) GetSummaryByID(ctx context.Context, id uuid.UUID) (*entity.DocumentSummary, error) {
	var summary entity.DocumentSummary
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &summary, nil
}

func (r *DocumentSummaryRepository) DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&entity.DocumentSummary{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
