package repository

import (
	"context"
	"errors"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITTSHistoryRepository interface {
	CreateTTSHistory(ctx context.Context, item *entity.TTSHistory) error
	GetTTSHistoryByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.TTSHistory, int64, error)
	GetTTSHistoryByID(ctx context.Context, id uuid.UUID) (*entity.TTSHistory, error)
	DeleteTTSHistory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type TTSHistoryRepository struct {
	db *gorm.DB
}

func NewTTSHistoryRepository(db *gorm.DB) *TTSHistoryRepository {
	return &TTSHistoryRepository{db: db}
}

func (r *TTSHistoryRepository) CreateTTSHistory(ctx context.Context, item *entity.TTSHistory) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *TTSHistoryRepository) GetTTSHistoryByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.TTSHistory, int64, error) {
	var items []entity.TTSHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.TTSHistory{})
	if userID == uuid.Nil {
		query = query.Where("user_id IS NULL")
	} else {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Order("created_at DESC").
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *TTSHistoryRepository) GetTTSHistoryByID(ctx context.Context, id uuid.UUID) (*entity.TTSHistory, error) {
	var item entity.TTSHistory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *TTSHistoryRepository) DeleteTTSHistory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if userID != uuid.Nil {
		query = query.Where("user_id = ? OR user_id IS NULL", userID)
	}
	result := query.Delete(&entity.TTSHistory{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
