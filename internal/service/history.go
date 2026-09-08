package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"

	"github.com/google/uuid"
)

type IHistoryService interface {
	GetAllHistory(ctx context.Context, userID uuid.UUID, query model.UnifiedHistoryQuery) (*model.UnifiedHistoryResponse, error)
	GetHistoryStats(ctx context.Context, userID uuid.UUID) (*model.HistoryStatsResponse, error)
	DeleteHistoryItem(ctx context.Context, userID uuid.UUID, itemType string, id uuid.UUID) error
	ClearAllHistory(ctx context.Context, userID uuid.UUID, historyType string) error
}

type HistoryService struct {
	historyRepo         repository.IHistoryRepository
	summaryRepo         repository.IDocumentSummaryRepository
	transcriptionRepo   repository.ITranscriptionRepository
	ttsRepo             repository.ITTSHistoryRepository
}

func NewHistoryService(
	historyRepo repository.IHistoryRepository,
	summaryRepo repository.IDocumentSummaryRepository,
	transcriptionRepo repository.ITranscriptionRepository,
	ttsRepo repository.ITTSHistoryRepository,
) *HistoryService {
	return &HistoryService{
		historyRepo:       historyRepo,
		summaryRepo:       summaryRepo,
		transcriptionRepo: transcriptionRepo,
		ttsRepo:           ttsRepo,
	}
}

func (s *HistoryService) GetAllHistory(ctx context.Context, userID uuid.UUID, query model.UnifiedHistoryQuery) (*model.UnifiedHistoryResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	counts, _, _, err := s.historyRepo.GetHistoryCounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history counts: %w", err)
	}

	filterType := strings.ToLower(strings.TrimSpace(query.Type))

	var items []model.UnifiedHistoryItem
	var totalRecords int64

	switch filterType {
	case "summary", "document_summary", "documents":
		summaries, total, err := s.historyRepo.GetDocumentSummaries(ctx, userID, query.Search, query.Limit, offset)
		if err != nil {
			return nil, err
		}
		totalRecords = total
		for _, sum := range summaries {
			items = append(items, convertSummaryToItem(sum))
		}

	case "stt", "transcription", "transcriptions", "speech":
		transcriptions, total, err := s.historyRepo.GetTranscriptions(ctx, userID, query.Search, query.Limit, offset)
		if err != nil {
			return nil, err
		}
		totalRecords = total
		for _, tr := range transcriptions {
			items = append(items, convertTranscriptionToItem(tr))
		}

	case "tts", "text_to_speech":
		ttsList, total, err := s.historyRepo.GetTTSHistories(ctx, userID, query.Search, query.Limit, offset)
		if err != nil {
			return nil, err
		}
		totalRecords = total
		for _, t := range ttsList {
			items = append(items, convertTTSToItem(t))
		}

	default: // "all" or empty
		// Fetch top items across all categories for chronological merging
		// To paginate accurately across multiple tables, fetch up to offset+limit from each
		fetchLimit := offset + query.Limit

		summaries, sumTotal, err := s.historyRepo.GetDocumentSummaries(ctx, userID, query.Search, fetchLimit, 0)
		if err != nil {
			return nil, err
		}

		transcriptions, sttTotal, err := s.historyRepo.GetTranscriptions(ctx, userID, query.Search, fetchLimit, 0)
		if err != nil {
			return nil, err
		}

		ttsList, ttsTotal, err := s.historyRepo.GetTTSHistories(ctx, userID, query.Search, fetchLimit, 0)
		if err != nil {
			return nil, err
		}

		totalRecords = sumTotal + sttTotal + ttsTotal

		var combined []model.UnifiedHistoryItem
		for _, sum := range summaries {
			combined = append(combined, convertSummaryToItem(sum))
		}
		for _, tr := range transcriptions {
			combined = append(combined, convertTranscriptionToItem(tr))
		}
		for _, t := range ttsList {
			combined = append(combined, convertTTSToItem(t))
		}

		// Sort merged items by CreatedAt DESC
		sort.Slice(combined, func(i, j int) bool {
			return combined[i].CreatedAt.After(combined[j].CreatedAt)
		})

		// Apply pagination window
		if offset < len(combined) {
			end := offset + query.Limit
			if end > len(combined) {
				end = len(combined)
			}
			items = combined[offset:end]
		} else {
			items = []model.UnifiedHistoryItem{}
		}
	}

	if items == nil {
		items = []model.UnifiedHistoryItem{}
	}

	return &model.UnifiedHistoryResponse{
		Data:   items,
		Total:  totalRecords,
		Counts: counts,
		Pagination: model.Pagination{
			Page:  query.Page,
			Limit: query.Limit,
		},
	}, nil
}

func (s *HistoryService) GetHistoryStats(ctx context.Context, userID uuid.UUID) (*model.HistoryStatsResponse, error) {
	counts, sttDur, ttsDur, err := s.historyRepo.GetHistoryCounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history stats: %w", err)
	}

	return &model.HistoryStatsResponse{
		Counts:              counts,
		TotalSpeechDuration: sttDur,
		TotalTTSDuration:    ttsDur,
	}, nil
}

func (s *HistoryService) DeleteHistoryItem(ctx context.Context, userID uuid.UUID, itemType string, id uuid.UUID) error {
	normType := strings.ToLower(strings.TrimSpace(itemType))
	switch normType {
	case "summary", "document_summary", "documents":
		return s.summaryRepo.DeleteSummary(ctx, id, userID)
	case "stt", "transcription", "transcriptions", "speech":
		return s.transcriptionRepo.DeleteTranscription(ctx, id, userID)
	case "tts", "text_to_speech":
		return s.ttsRepo.DeleteTTSHistory(ctx, id, userID)
	default:
		return fmt.Errorf("unsupported history type: %s", itemType)
	}
}

func (s *HistoryService) ClearAllHistory(ctx context.Context, userID uuid.UUID, historyType string) error {
	return s.historyRepo.DeleteAllByUserID(ctx, userID, historyType)
}

// Helpers for unified mapping
func convertSummaryToItem(s entity.DocumentSummary) model.UnifiedHistoryItem {
	title := s.Title
	if title == "" {
		title = s.FileName
	}
	desc := s.Summary
	if len(desc) > 200 {
		desc = desc[:200] + "..."
	}

	var keyPoints []string
	if s.KeyPoints != "" {
		_ = json.Unmarshal([]byte(s.KeyPoints), &keyPoints)
	}

	return model.UnifiedHistoryItem{
		ID:          s.ID,
		Type:        model.HistoryTypeDocumentSummary,
		Title:       title,
		Content:     s.Summary,
		Description: desc,
		Language:    s.Language,
		CreatedAt:   s.CreatedAt,
		Details: model.DocumentSummaryResponse{
			ID:             s.ID,
			FileName:       s.FileName,
			FileType:       s.FileType,
			FileSize:       s.FileSize,
			Title:          s.Title,
			Summary:        s.Summary,
			KeyPoints:      keyPoints,
			Explanation:    s.Explanation,
			Language:       s.Language,
			DetailLevel:    s.DetailLevel,
			TargetAudience: s.TargetAudience,
			TokenCount:     s.TokenCount,
			CreatedAt:      s.CreatedAt,
		},
	}
}

func convertTranscriptionToItem(t entity.Transcription) model.UnifiedHistoryItem {
	title := "Speech Transcription"
	desc := t.Text
	if len(desc) > 200 {
		desc = desc[:200] + "..."
	}

	return model.UnifiedHistoryItem{
		ID:          t.ID,
		Type:        model.HistoryTypeSTT,
		Title:       title,
		Content:     t.Text,
		Description: desc,
		Language:    t.Language,
		DurationMs:  t.DurationMs,
		CreatedAt:   t.CreatedAt,
		Details: model.TranscriptionResponse{
			ID:         t.ID,
			UserID:     t.UserID,
			SessionID:  t.SessionID,
			Language:   t.Language,
			Text:       t.Text,
			Confidence: t.Confidence,
			DurationMs: t.DurationMs,
			CreatedAt:  t.CreatedAt,
		},
	}
}

func convertTTSToItem(t entity.TTSHistory) model.UnifiedHistoryItem {
	title := fmt.Sprintf("Text-to-Speech (%s)", t.Voice)
	desc := t.Text
	if len(desc) > 200 {
		desc = desc[:200] + "..."
	}

	return model.UnifiedHistoryItem{
		ID:          t.ID,
		Type:        model.HistoryTypeTTS,
		Title:       title,
		Content:     t.Text,
		Description: desc,
		DurationMs:  t.DurationMs,
		CreatedAt:   t.CreatedAt,
		Details: model.TTSHistoryResponse{
			ID:         t.ID,
			UserID:     t.UserID,
			Text:       t.Text,
			Voice:      t.Voice,
			Format:     t.Format,
			DurationMs: t.DurationMs,
			CreatedAt:  t.CreatedAt,
		},
	}
}
