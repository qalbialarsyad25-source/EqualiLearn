package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/export"
	"EquiliLearn/pkg/gemini"
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

	// Speech Summarization (Gemini AI)
	SummarizeTranscription(ctx context.Context, req model.SummarizeSpeechRequest, userID *uuid.UUID) (*model.SpeechSummaryResponse, error)
	SummarizeSpeechAudio(ctx context.Context, req model.SummarizeSpeechAudioRequest, audioBytes []byte, filename string, contentType string, userID *uuid.UUID) (*model.SpeechSummaryResponse, error)
	ExportSpeechSummary(ctx context.Context, id uuid.UUID, format string) (*model.ExportFileResult, error)
	ExportSpeechSummaryDirect(ctx context.Context, req model.SummarizeSpeechRequest, format string, userID *uuid.UUID) (*model.ExportFileResult, error)

	// Text-to-Speech (TTS)
	SynthesizeSpeech(ctx context.Context, req model.SynthesizeSpeechRequest, userID *uuid.UUID) (*model.TTSAudioOutput, error)
	GetAvailableVoices(ctx context.Context) []model.TTSVoiceResponse
	GetTTSHistory(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.TTSHistoryResponse, int64, error)
	DeleteTTSHistory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type SpeechService struct {
	sttClient         stt.ISTTClient
	ttsClient         tts.ITTSClient
	transcriptionRepo repository.ITranscriptionRepository
	ttsHistoryRepo    repository.ITTSHistoryRepository
	geminiClient      gemini.IGeminiClient
	summaryRepo       repository.IDocumentSummaryRepository
}

func NewSpeechService(
	sttClient stt.ISTTClient,
	ttsClient tts.ITTSClient,
	transcriptionRepo repository.ITranscriptionRepository,
	ttsHistoryRepo repository.ITTSHistoryRepository,
	geminiClient gemini.IGeminiClient,
	summaryRepo repository.IDocumentSummaryRepository,
) *SpeechService {
	return &SpeechService{
		sttClient:         sttClient,
		ttsClient:         ttsClient,
		transcriptionRepo: transcriptionRepo,
		ttsHistoryRepo:    ttsHistoryRepo,
		geminiClient:      geminiClient,
		summaryRepo:       summaryRepo,
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
// Speech Summarization (Gemini AI)
// ==========================================

func (s *SpeechService) SummarizeTranscription(ctx context.Context, req model.SummarizeSpeechRequest, userID *uuid.UUID) (*model.SpeechSummaryResponse, error) {
	var transcriptText string
	var lang string = req.Language

	if req.TranscriptionID != nil {
		if s.transcriptionRepo == nil {
			return nil, fmt.Errorf("transcription repository not available")
		}
		tr, err := s.transcriptionRepo.GetTranscriptionByID(ctx, *req.TranscriptionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get transcription: %w", err)
		}
		if tr == nil {
			return nil, fmt.Errorf("transcription not found")
		}
		transcriptText = tr.Text
		if lang == "" {
			lang = tr.Language
		}
	} else if strings.TrimSpace(req.Text) != "" {
		transcriptText = strings.TrimSpace(req.Text)
	} else {
		return nil, fmt.Errorf("either transcription_id or text is required")
	}

	if lang == "" {
		lang = "id"
	}

	detailLevel := req.DetailLevel
	if detailLevel == "" {
		detailLevel = "balanced"
	}

	targetAudience := req.TargetAudience
	if targetAudience == "" {
		targetAudience = "student"
	}

	title := req.Title
	if title == "" {
		title = "Speech Transcript Summary"
	}

	if s.geminiClient == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	geminiReq := gemini.GeminiSummaryRequest{
		TextContent:    transcriptText,
		FileName:       title,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
	}

	result, err := s.geminiClient.SummarizeContent(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("ai speech summarization failed: %w", err)
	}

	summaryID := uuid.New()
	keyPointsJSON, _ := json.Marshal(result.KeyPoints)

	shouldSave := true
	if req.SaveToHistory != nil {
		shouldSave = *req.SaveToHistory
	}

	if shouldSave && s.summaryRepo != nil {
		summaryEntity := &entity.DocumentSummary{
			ID:             summaryID,
			UserID:         userID,
			FileName:       title,
			FileType:       "speech",
			FileSize:       int64(len(transcriptText)),
			Title:          result.Title,
			Summary:        result.Summary,
			KeyPoints:      string(keyPointsJSON),
			Explanation:    result.Explanation,
			Language:       lang,
			DetailLevel:    detailLevel,
			TargetAudience: targetAudience,
			TokenCount:     result.TokenCount,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.summaryRepo.CreateDocumentSummary(ctx, summaryEntity); err != nil {
			return nil, fmt.Errorf("failed to persist speech summary to database: %w", err)
		}
	}

	return &model.SpeechSummaryResponse{
		ID:              summaryID,
		UserID:          userID,
		TranscriptionID: req.TranscriptionID,
		Title:           result.Title,
		TranscriptText:  transcriptText,
		Summary:         result.Summary,
		KeyPoints:       result.KeyPoints,
		Explanation:     result.Explanation,
		Language:        lang,
		DetailLevel:     detailLevel,
		TargetAudience:  targetAudience,
		Model:           result.Model,
		TokenCount:      result.TokenCount,
		CreatedAt:       time.Now(),
	}, nil
}

func (s *SpeechService) SummarizeSpeechAudio(ctx context.Context, req model.SummarizeSpeechAudioRequest, audioBytes []byte, filename string, contentType string, userID *uuid.UUID) (*model.SpeechSummaryResponse, error) {
	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("audio file is empty")
	}

	lang := req.Language
	if lang == "" {
		lang = "id"
	}

	detailLevel := req.DetailLevel
	if detailLevel == "" {
		detailLevel = "balanced"
	}

	targetAudience := req.TargetAudience
	if targetAudience == "" {
		targetAudience = "student"
	}

	title := req.Title
	if title == "" {
		title = filename
		if title == "" {
			title = "Audio Recording"
		}
	}

	if contentType == "" {
		contentType = "audio/mp3"
	}

	if s.geminiClient == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	geminiReq := gemini.GeminiSummaryRequest{
		InlineData:     audioBytes,
		MIMEType:       contentType,
		FileName:       filename,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
	}

	result, err := s.geminiClient.SummarizeContent(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("ai audio summarization failed: %w", err)
	}

	summaryID := uuid.New()
	keyPointsJSON, _ := json.Marshal(result.KeyPoints)

	shouldSave := true
	if req.SaveToHistory != nil {
		shouldSave = *req.SaveToHistory
	}

	if shouldSave && s.summaryRepo != nil {
		summaryEntity := &entity.DocumentSummary{
			ID:             summaryID,
			UserID:         userID,
			FileName:       filename,
			FileType:       "audio",
			FileSize:       int64(len(audioBytes)),
			Title:          result.Title,
			Summary:        result.Summary,
			KeyPoints:      string(keyPointsJSON),
			Explanation:    result.Explanation,
			Language:       lang,
			DetailLevel:    detailLevel,
			TargetAudience: targetAudience,
			TokenCount:     result.TokenCount,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.summaryRepo.CreateDocumentSummary(ctx, summaryEntity); err != nil {
			return nil, fmt.Errorf("failed to persist audio summary to database: %w", err)
		}
	}

	return &model.SpeechSummaryResponse{
		ID:             summaryID,
		UserID:         userID,
		Title:          result.Title,
		TranscriptText: "",
		Summary:        result.Summary,
		KeyPoints:      result.KeyPoints,
		Explanation:    result.Explanation,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
		Model:          result.Model,
		TokenCount:     result.TokenCount,
		CreatedAt:      time.Now(),
	}, nil
}

func (s *SpeechService) ExportSpeechSummary(ctx context.Context, id uuid.UUID, format string) (*model.ExportFileResult, error) {
	if s.summaryRepo == nil {
		return nil, fmt.Errorf("summary repository not available")
	}

	summaryEntity, err := s.summaryRepo.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve speech summary: %w", err)
	}
	if summaryEntity == nil {
		return nil, fmt.Errorf("speech summary not found")
	}

	var keyPoints []string
	if summaryEntity.KeyPoints != "" {
		_ = json.Unmarshal([]byte(summaryEntity.KeyPoints), &keyPoints)
	}

	payload := export.ExportPayload{
		Title:          summaryEntity.Title,
		Summary:        summaryEntity.Summary,
		KeyPoints:      keyPoints,
		Explanation:    summaryEntity.Explanation,
		Language:       summaryEntity.Language,
		DetailLevel:    summaryEntity.DetailLevel,
		TargetAudience: summaryEntity.TargetAudience,
		SourceType:     "Speech-to-Text Transcript",
		CreatedAt:      summaryEntity.CreatedAt,
	}

	normFormat := strings.ToLower(strings.TrimSpace(format))
	baseName := strings.ReplaceAll(summaryEntity.Title, " ", "_")
	if baseName == "" {
		baseName = "speech_summary"
	}
	baseName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, baseName)

	switch normFormat {
	case "pdf", "application/pdf":
		pdfBytes := export.GeneratePDF(payload)
		return &model.ExportFileResult{
			Data:        pdfBytes,
			Filename:    fmt.Sprintf("%s.pdf", baseName),
			ContentType: "application/pdf",
		}, nil
	case "md", "markdown", "text/markdown":
		mdBytes := export.GenerateMarkdown(payload)
		return &model.ExportFileResult{
			Data:        mdBytes,
			Filename:    fmt.Sprintf("%s.md", baseName),
			ContentType: "text/markdown; charset=utf-8",
		}, nil
	case "txt", "text", "text/plain", "":
		txtBytes := export.GenerateText(payload)
		return &model.ExportFileResult{
			Data:        txtBytes,
			Filename:    fmt.Sprintf("%s.txt", baseName),
			ContentType: "text/plain; charset=utf-8",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported export format %q. Supported formats: 'pdf', 'txt', 'md'", format)
	}
}

func (s *SpeechService) ExportSpeechSummaryDirect(ctx context.Context, req model.SummarizeSpeechRequest, format string, userID *uuid.UUID) (*model.ExportFileResult, error) {
	summaryResp, err := s.SummarizeTranscription(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	payload := export.ExportPayload{
		Title:          summaryResp.Title,
		Summary:        summaryResp.Summary,
		KeyPoints:      summaryResp.KeyPoints,
		Explanation:    summaryResp.Explanation,
		TranscriptText: summaryResp.TranscriptText,
		Language:       summaryResp.Language,
		DetailLevel:    summaryResp.DetailLevel,
		TargetAudience: summaryResp.TargetAudience,
		SourceType:     "Speech-to-Text Transcript",
		CreatedAt:      summaryResp.CreatedAt,
	}

	normFormat := strings.ToLower(strings.TrimSpace(format))
	baseName := strings.ReplaceAll(summaryResp.Title, " ", "_")
	if baseName == "" {
		baseName = "speech_summary"
	}
	baseName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, baseName)

	switch normFormat {
	case "pdf", "application/pdf":
		pdfBytes := export.GeneratePDF(payload)
		return &model.ExportFileResult{
			Data:        pdfBytes,
			Filename:    fmt.Sprintf("%s.pdf", baseName),
			ContentType: "application/pdf",
		}, nil
	case "md", "markdown", "text/markdown":
		mdBytes := export.GenerateMarkdown(payload)
		return &model.ExportFileResult{
			Data:        mdBytes,
			Filename:    fmt.Sprintf("%s.md", baseName),
			ContentType: "text/markdown; charset=utf-8",
		}, nil
	case "txt", "text", "text/plain", "":
		txtBytes := export.GenerateText(payload)
		return &model.ExportFileResult{
			Data:        txtBytes,
			Filename:    fmt.Sprintf("%s.txt", baseName),
			ContentType: "text/plain; charset=utf-8",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported export format %q. Supported formats: 'pdf', 'txt', 'md'", format)
	}
}

// ==========================================
// Text-to-Speech (TTS) Implementations
// ==========================================

func (s *SpeechService) SynthesizeSpeech(ctx context.Context, req model.SynthesizeSpeechRequest, userID *uuid.UUID) (*model.TTSAudioOutput, error) {
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

	// Persist TTS generation history if user is authenticated and repository is available
	if userID != nil && s.ttsHistoryRepo != nil {
		historyRecord := &entity.TTSHistory{
			ID:         uuid.New(),
			UserID:     userID,
			Text:       trimmedText,
			Voice:      voice,
			Format:     format,
			DurationMs: estimatedDurationMs,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		_ = s.ttsHistoryRepo.CreateTTSHistory(ctx, historyRecord)
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

func (s *SpeechService) GetTTSHistory(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.TTSHistoryResponse, int64, error) {
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	if s.ttsHistoryRepo == nil {
		return []model.TTSHistoryResponse{}, 0, nil
	}

	records, total, err := s.ttsHistoryRepo.GetTTSHistoryByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]model.TTSHistoryResponse, 0, len(records))
	for _, r := range records {
		responses = append(responses, model.TTSHistoryResponse{
			ID:         r.ID,
			UserID:     r.UserID,
			Text:       r.Text,
			Voice:      r.Voice,
			Format:     r.Format,
			DurationMs: r.DurationMs,
			CreatedAt:  r.CreatedAt,
		})
	}

	return responses, total, nil
}

func (s *SpeechService) DeleteTTSHistory(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if s.ttsHistoryRepo == nil {
		return nil
	}
	return s.ttsHistoryRepo.DeleteTTSHistory(ctx, id, userID)
}
