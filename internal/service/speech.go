package service

import (
	"context"
	"fmt"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/stt"

	"github.com/google/uuid"
)

type ISpeechService interface {
	StartSTTSession(ctx context.Context, userID uuid.UUID, cfg model.STTStreamConfig) (stt.ISTTSession, string, error)
	SaveFinalTranscription(ctx context.Context, userID uuid.UUID, sessionID string, language string, text string, confidence float64, durationMs int64) (*entity.Transcription, error)
	GetTranscriptionHistory(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.TranscriptionResponse, error)
	DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type SpeechService struct {
	sttClient         stt.ISTTClient
	transcriptionRepo repository.ITranscriptionRepository
}

func NewSpeechService(sttClient stt.ISTTClient, transcriptionRepo repository.ITranscriptionRepository) *SpeechService {
	return &SpeechService{
		sttClient:         sttClient,
		transcriptionRepo: transcriptionRepo,
	}
}

func (s *SpeechService) StartSTTSession(ctx context.Context, userID uuid.UUID, cfg model.STTStreamConfig) (stt.ISTTSession, string, error) {
	sessionID := uuid.New().String()

	lang := cfg.LanguageCode
	if lang == "" {
		lang = "id-ID"
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	encoding := cfg.Encoding
	if encoding == "" {
		encoding = "linear16"
	}

	sttConfig := stt.STTConfig{
		LanguageCode:   lang,
		SampleRate:     sampleRate,
		Encoding:       encoding,
		InterimResults: cfg.InterimResults,
	}

	session, err := s.sttClient.StartStream(ctx, sttConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize STT session: %w", err)
	}

	return session, sessionID, nil
}

func (s *SpeechService) SaveFinalTranscription(ctx context.Context, userID uuid.UUID, sessionID string, language string, text string, confidence float64, durationMs int64) (*entity.Transcription, error) {
	if text == "" {
		return nil, nil
	}

	record := &entity.Transcription{
		ID:         uuid.New(),
		UserID:     userID,
		SessionID:  sessionID,
		Language:   language,
		Text:       text,
		Confidence: confidence,
		DurationMs: durationMs,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := s.transcriptionRepo.CreateTranscription(ctx, record)
	if err != nil {
		return nil, fmt.Errorf("failed to save transcription: %w", err)
	}

	return record, nil
}

func (s *SpeechService) GetTranscriptionHistory(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.TranscriptionResponse, error) {
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	records, err := s.transcriptionRepo.GetTranscriptionsByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, err
	}

	var responses []model.TranscriptionResponse
	for _, r := range records {
		responses = append(responses, model.TranscriptionResponse{
			ID:         r.ID,
			UserID:     r.UserID,
			SessionID:  r.SessionID,
			Language:   r.Language,
			Text:       r.Text,
			Confidence: r.Confidence,
			DurationMs: r.DurationMs,
			CreatedAt:  r.CreatedAt,
		})
	}

	return responses, nil
}

func (s *SpeechService) DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.transcriptionRepo.DeleteTranscription(ctx, id, userID)
}
