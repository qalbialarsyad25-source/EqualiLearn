package repository

import (
	"context"
	"fmt"
	"strings"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IHistoryRepository interface {
	GetHistoryCounts(ctx context.Context, userID uuid.UUID) (model.HistoryCounts, int64, int64, error)
	GetDocumentSummaries(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.DocumentSummary, int64, error)
	GetTranscriptions(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.Transcription, int64, error)
	GetTTSHistories(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.TTSHistory, int64, error)
	DeleteAllByUserID(ctx context.Context, userID uuid.UUID, historyType string) error
}

type HistoryRepository struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) GetHistoryCounts(ctx context.Context, userID uuid.UUID) (model.HistoryCounts, int64, int64, error) {
	var counts model.HistoryCounts
	var totalSTTDuration int64
	var totalTTSDuration int64

	// Count summaries
	if err := r.db.WithContext(ctx).Model(&entity.DocumentSummary{}).Where("user_id = ?", userID).Count(&counts.TotalSummaries).Error; err != nil {
		return counts, 0, 0, err
	}

	// Count STT
	if err := r.db.WithContext(ctx).Model(&entity.Transcription{}).Where("user_id = ?", userID).Count(&counts.TotalSTT).Error; err != nil {
		return counts, 0, 0, err
	}

	// Count TTS
	if err := r.db.WithContext(ctx).Model(&entity.TTSHistory{}).Where("user_id = ?", userID).Count(&counts.TotalTTS).Error; err != nil {
		return counts, 0, 0, err
	}

	counts.TotalAll = counts.TotalSummaries + counts.TotalSTT + counts.TotalTTS

	// Durations
	var sttDur struct{ Total int64 }
	r.db.WithContext(ctx).Model(&entity.Transcription{}).Where("user_id = ?", userID).Select("COALESCE(SUM(duration_ms), 0) as total").Scan(&sttDur)
	totalSTTDuration = sttDur.Total

	var ttsDur struct{ Total int64 }
	r.db.WithContext(ctx).Model(&entity.TTSHistory{}).Where("user_id = ?", userID).Select("COALESCE(SUM(duration_ms), 0) as total").Scan(&ttsDur)
	totalTTSDuration = ttsDur.Total

	return counts, totalSTTDuration, totalTTSDuration, nil
}

func (r *HistoryRepository) GetDocumentSummaries(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.DocumentSummary, int64, error) {
	var summaries []entity.DocumentSummary
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.DocumentSummary{}).Where("user_id = ?", userID)
	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(file_name) LIKE ? OR LOWER(summary) LIKE ?", s, s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&summaries).Error
	if err != nil {
		return nil, 0, err
	}

	return summaries, total, nil
}

func (r *HistoryRepository) GetTranscriptions(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.Transcription, int64, error) {
	var transcriptions []entity.Transcription
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Transcription{}).Where("user_id = ?", userID)
	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(text) LIKE ? OR LOWER(session_id) LIKE ?", s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&transcriptions).Error
	if err != nil {
		return nil, 0, err
	}

	return transcriptions, total, nil
}

func (r *HistoryRepository) GetTTSHistories(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.TTSHistory, int64, error) {
	var items []entity.TTSHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.TTSHistory{}).Where("user_id = ?", userID)
	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(text) LIKE ? OR LOWER(voice) LIKE ?", s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Order("created_at DESC").Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *HistoryRepository) DeleteAllByUserID(ctx context.Context, userID uuid.UUID, historyType string) error {
	normType := strings.ToLower(strings.TrimSpace(historyType))
	switch normType {
	case "summary", "document_summary", "documents":
		return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.DocumentSummary{}).Error
	case "stt", "transcription", "transcriptions", "speech":
		return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.Transcription{}).Error
	case "tts", "text_to_speech":
		return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.TTSHistory{}).Error
	case "", "all":
		if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.DocumentSummary{}).Error; err != nil {
			return err
		}
		if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.Transcription{}).Error; err != nil {
			return err
		}
		if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.TTSHistory{}).Error; err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown history type: %s", historyType)
	}
}
