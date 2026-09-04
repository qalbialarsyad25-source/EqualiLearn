package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/stt"
	"EquiliLearn/pkg/tts"

	"github.com/google/uuid"
)

type ISpeechService interface {
	// Speech-to-Text (STT)
	StartSTTSession(ctx context.Context, userID *uuid.UUID, cfg model.STTStreamConfig) (stt.ISTTSession, string, error)
	SaveFinalTranscription(ctx context.Context, userID *uuid.UUID, sessionID string, language string, text string, confidence float64, durationMs int64) (*entity.Transcription, error)
	GetTranscriptionHistory(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.TranscriptionResponse, error)
	DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// Text-to-Speech (TTS)
	SynthesizeSpeech(ctx context.Context, req model.SynthesizeSpeechRequest) (*model.TTSAudioOutput, error)
	GetAvailableVoices(ctx context.Context) []model.TTSVoiceResponse
}

type SpeechService struct {
	sttClient         stt.ISTTClient
	ttsClient         tts.ITTSClient
	transcriptionRepo repository.ITranscriptionRepository
}

func NewSpeechService(sttClient stt.ISTTClient, ttsClient tts.ITTSClient, transcriptionRepo repository.ITranscriptionRepository) *SpeechService {
	return &SpeechService{
		sttClient:         sttClient,
		ttsClient:         ttsClient,
		transcriptionRepo: transcriptionRepo,
	}
}

// ==========================================
// Speech-to-Text (STT) Implementations
// ==========================================

func (s *SpeechService) StartSTTSession(ctx context.Context, userID *uuid.UUID, cfg model.STTStreamConfig) (stt.ISTTSession, string, error) {
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

func (s *SpeechService) SaveFinalTranscription(ctx context.Context, userID *uuid.UUID, sessionID string, language string, text string, confidence float64, durationMs int64) (*entity.Transcription, error) {
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

// ==========================================
// Text-to-Speech (TTS) Implementations
// ==========================================

func (s *SpeechService) SynthesizeSpeech(ctx context.Context, req model.SynthesizeSpeechRequest) (*model.TTSAudioOutput, error) {
	trimmedText := strings.TrimSpace(req.Text)
	if trimmedText == "" {
		return nil, fmt.Errorf("synthesis text cannot be empty")
	}

	voice := req.Voice
	if voice == "" {
		voice = "aura-asteria-en"
	}

	format := strings.ToLower(req.Format)
	if format == "" {
		format = "mp3"
	}

	container := ""
	if format == "wav" {
		container = "wav"
	}

	ttsReq := tts.TTSRequest{
		Text:       trimmedText,
		Model:      voice,
		Encoding:   format,
		Container:  container,
		SampleRate: req.SampleRate,
	}

	result, err := s.ttsClient.Synthesize(ctx, ttsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	// Approximate speech duration based on word count (~150 words per minute)
	wordCount := len(strings.Fields(trimmedText))
	estimatedDurationMs := int64((float64(wordCount) / 150.0) * 60.0 * 1000.0)
	if estimatedDurationMs < 500 {
		estimatedDurationMs = 500
	}

	return &model.TTSAudioOutput{
		AudioData:          result.AudioBytes,
		ContentType:        result.ContentType,
		Voice:              result.Model,
		Format:             result.Format,
		DurationEstimateMs: estimatedDurationMs,
	}, nil
}

func (s *SpeechService) GetAvailableVoices(ctx context.Context) []model.TTSVoiceResponse {
	voices := s.ttsClient.GetVoices()
	responses := make([]model.TTSVoiceResponse, 0, len(voices))
	for _, v := range voices {
		responses = append(responses, model.TTSVoiceResponse{
			ID:          v.ID,
			Name:        v.Name,
			Gender:      v.Gender,
			Language:    v.Language,
			Description: v.Description,
			SampleRate:  v.SampleRate,
		})
	}
	return responses
}
