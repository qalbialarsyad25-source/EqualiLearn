package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
)

type mockHistoryRepo struct {
	summaries      []entity.DocumentSummary
	transcriptions []entity.Transcription
	ttsList        []entity.TTSHistory
}

func (m *mockHistoryRepo) GetHistoryCounts(ctx context.Context, userID uuid.UUID) (model.HistoryCounts, int64, int64, error) {
	var counts model.HistoryCounts
	var sttDur int64
	var ttsDur int64

	for _, s := range m.summaries {
		if s.UserID != nil && *s.UserID == userID {
			counts.TotalSummaries++
		}
	}
	for _, t := range m.transcriptions {
		if t.UserID != nil && *t.UserID == userID {
			counts.TotalSTT++
			sttDur += t.DurationMs
		}
	}
	for _, t := range m.ttsList {
		if t.UserID != nil && *t.UserID == userID {
			counts.TotalTTS++
			ttsDur += t.DurationMs
		}
	}
	counts.TotalAll = counts.TotalSummaries + counts.TotalSTT + counts.TotalTTS

	return counts, sttDur, ttsDur, nil
}

func (m *mockHistoryRepo) GetDocumentSummaries(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.DocumentSummary, int64, error) {
	var results []entity.DocumentSummary
	for _, s := range m.summaries {
		if s.UserID != nil && *s.UserID == userID {
			if search != "" {
				sLower := strings.ToLower(search)
				if !strings.Contains(strings.ToLower(s.Title), sLower) &&
					!strings.Contains(strings.ToLower(s.FileName), sLower) &&
					!strings.Contains(strings.ToLower(s.Summary), sLower) {
					continue
				}
			}
			results = append(results, s)
		}
	}

	total := int64(len(results))
	if offset > len(results) {
		return []entity.DocumentSummary{}, total, nil
	}
	end := offset + limit
	if end > len(results) || limit <= 0 {
		end = len(results)
	}
	return results[offset:end], total, nil
}

func (m *mockHistoryRepo) GetTranscriptions(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.Transcription, int64, error) {
	var results []entity.Transcription
	for _, t := range m.transcriptions {
		if t.UserID != nil && *t.UserID == userID {
			if search != "" {
				sLower := strings.ToLower(search)
				if !strings.Contains(strings.ToLower(t.Text), sLower) &&
					!strings.Contains(strings.ToLower(t.SessionID), sLower) {
					continue
				}
			}
			results = append(results, t)
		}
	}

	total := int64(len(results))
	if offset > len(results) {
		return []entity.Transcription{}, total, nil
	}
	end := offset + limit
	if end > len(results) || limit <= 0 {
		end = len(results)
	}
	return results[offset:end], total, nil
}

func (m *mockHistoryRepo) GetTTSHistories(ctx context.Context, userID uuid.UUID, search string, limit, offset int) ([]entity.TTSHistory, int64, error) {
	var results []entity.TTSHistory
	for _, t := range m.ttsList {
		if t.UserID != nil && *t.UserID == userID {
			if search != "" {
				sLower := strings.ToLower(search)
				if !strings.Contains(strings.ToLower(t.Text), sLower) &&
					!strings.Contains(strings.ToLower(t.Voice), sLower) {
					continue
				}
			}
			results = append(results, t)
		}
	}

	total := int64(len(results))
	if offset > len(results) {
		return []entity.TTSHistory{}, total, nil
	}
	end := offset + limit
	if end > len(results) || limit <= 0 {
		end = len(results)
	}
	return results[offset:end], total, nil
}

func (m *mockHistoryRepo) DeleteAllByUserID(ctx context.Context, userID uuid.UUID, historyType string) error {
	norm := strings.ToLower(strings.TrimSpace(historyType))
	switch norm {
	case "summary", "document_summary":
		var rem []entity.DocumentSummary
		for _, s := range m.summaries {
			if !(s.UserID != nil && *s.UserID == userID) {
				rem = append(rem, s)
			}
		}
		m.summaries = rem
	case "stt", "transcription":
		var rem []entity.Transcription
		for _, t := range m.transcriptions {
			if !(t.UserID != nil && *t.UserID == userID) {
				rem = append(rem, t)
			}
		}
		m.transcriptions = rem
	case "tts":
		var rem []entity.TTSHistory
		for _, t := range m.ttsList {
			if !(t.UserID != nil && *t.UserID == userID) {
				rem = append(rem, t)
			}
		}
		m.ttsList = rem
	default:
		var remS []entity.DocumentSummary
		for _, s := range m.summaries {
			if !(s.UserID != nil && *s.UserID == userID) {
				remS = append(remS, s)
			}
		}
		m.summaries = remS

		var remT []entity.Transcription
		for _, t := range m.transcriptions {
			if !(t.UserID != nil && *t.UserID == userID) {
				remT = append(remT, t)
			}
		}
		m.transcriptions = remT

		var remTTS []entity.TTSHistory
		for _, t := range m.ttsList {
			if !(t.UserID != nil && *t.UserID == userID) {
				remTTS = append(remTTS, t)
			}
		}
		m.ttsList = remTTS
	}
	return nil
}

type mockSummaryRepo struct {
	summaries []entity.DocumentSummary
}

func (m *mockSummaryRepo) CreateDocumentSummary(ctx context.Context, s *entity.DocumentSummary) error {
	m.summaries = append(m.summaries, *s)
	return nil
}

func (m *mockSummaryRepo) GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.DocumentSummary, int64, error) {
	var res []entity.DocumentSummary
	for _, s := range m.summaries {
		if s.UserID != nil && *s.UserID == userID {
			res = append(res, s)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockSummaryRepo) GetSummaryByID(ctx context.Context, id uuid.UUID) (*entity.DocumentSummary, error) {
	for _, s := range m.summaries {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockSummaryRepo) DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	var rem []entity.DocumentSummary
	for _, s := range m.summaries {
		if !(s.ID == id && s.UserID != nil && *s.UserID == userID) {
			rem = append(rem, s)
		}
	}
	m.summaries = rem
	return nil
}

func TestHistoryService_GetAllHistory_Unified(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	summaryID := uuid.New()
	sttID := uuid.New()
	ttsID := uuid.New()

	summaries := []entity.DocumentSummary{
		{
			ID:        summaryID,
			UserID:    &userID,
			Title:     "Quantum Physics 101",
			FileName:  "quantum.pdf",
			Summary:   "Quantum mechanics summary notes.",
			CreatedAt: now.Add(-10 * time.Minute),
		},
	}

	transcriptions := []entity.Transcription{
		{
			ID:         sttID,
			UserID:     &userID,
			SessionID:  "session-1",
			Language:   "id-ID",
			Text:       "Diskusi mengenai fisika nuklir",
			DurationMs: 15000,
			CreatedAt:  now.Add(-5 * time.Minute),
		},
	}

	ttsList := []entity.TTSHistory{
		{
			ID:         ttsID,
			UserID:     &userID,
			Voice:      "aura-asteria-en",
			Text:       "Welcome to modern learning.",
			DurationMs: 3000,
			CreatedAt:  now.Add(-1 * time.Minute),
		},
	}

	hRepo := &mockHistoryRepo{
		summaries:      summaries,
		transcriptions: transcriptions,
		ttsList:        ttsList,
	}
	sRepo := &mockSummaryRepo{summaries: summaries}
	tRepo := &mockTranscriptionRepo{transcriptions: transcriptions}
	ttsRepo := &mockTTSHistoryRepo{items: ttsList}

	svc := NewHistoryService(hRepo, sRepo, tRepo, ttsRepo)
	ctx := context.Background()

	// 1. Get All History (Unified timeline, sorted by CreatedAt DESC)
	res, err := svc.GetAllHistory(ctx, userID, model.UnifiedHistoryQuery{Page: 1, Limit: 10, Type: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Total != 3 {
		t.Fatalf("expected total 3, got %d", res.Total)
	}
	if len(res.Data) != 3 {
		t.Fatalf("expected 3 items, got %d", len(res.Data))
	}

	// Verify order: newest first -> TTS (-1m), then STT (-5m), then Summary (-10m)
	if res.Data[0].Type != model.HistoryTypeTTS {
		t.Errorf("expected 1st item to be TTS, got %s", res.Data[0].Type)
	}
	if res.Data[1].Type != model.HistoryTypeSTT {
		t.Errorf("expected 2nd item to be STT, got %s", res.Data[1].Type)
	}
	if res.Data[2].Type != model.HistoryTypeDocumentSummary {
		t.Errorf("expected 3rd item to be document_summary, got %s", res.Data[2].Type)
	}

	// Verify Counts
	if res.Counts.TotalSummaries != 1 || res.Counts.TotalSTT != 1 || res.Counts.TotalTTS != 1 || res.Counts.TotalAll != 3 {
		t.Errorf("unexpected counts: %+v", res.Counts)
	}

	// 2. Filter by type = "summary"
	sumRes, err := svc.GetAllHistory(ctx, userID, model.UnifiedHistoryQuery{Page: 1, Limit: 10, Type: "summary"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sumRes.Data) != 1 || sumRes.Data[0].ID != summaryID {
		t.Errorf("expected summary item, got %+v", sumRes.Data)
	}

	// 3. Search query
	searchRes, err := svc.GetAllHistory(ctx, userID, model.UnifiedHistoryQuery{Page: 1, Limit: 10, Search: "fisika"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(searchRes.Data) != 1 || searchRes.Data[0].Type != model.HistoryTypeSTT {
		t.Errorf("expected STT search match for 'fisika', got %+v", searchRes.Data)
	}

	// 4. Get Statistics
	stats, err := svc.GetHistoryStats(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalSpeechDuration != 15000 || stats.TotalTTSDuration != 3000 {
		t.Errorf("unexpected durations in stats: %+v", stats)
	}

	// 5. Delete individual item
	err = svc.DeleteHistoryItem(ctx, userID, "tts", ttsID)
	if err != nil {
		t.Fatalf("failed to delete item: %v", err)
	}
	if len(ttsRepo.items) != 0 {
		t.Errorf("expected TTS repo to be empty after delete, got %d", len(ttsRepo.items))
	}

	// 6. Clear All History
	err = svc.ClearAllHistory(ctx, userID, "all")
	if err != nil {
		t.Fatalf("failed to clear history: %v", err)
	}
	if len(hRepo.summaries) != 0 || len(hRepo.transcriptions) != 0 || len(hRepo.ttsList) != 0 {
		t.Errorf("expected all repos to be cleared")
	}
}
