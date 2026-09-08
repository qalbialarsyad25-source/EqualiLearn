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
	"EquiliLearn/pkg/document"
	"EquiliLearn/pkg/export"
	"EquiliLearn/pkg/gemini"

	"github.com/google/uuid"
)

type IDocumentSummaryService interface {
	SummarizeDocument(ctx context.Context, req model.SummarizeDocumentRequest, fileBytes []byte, filename string, contentType string, userID *uuid.UUID) (*model.DocumentSummaryResponse, error)
	SummarizeText(ctx context.Context, req model.SummarizeTextRequest, userID *uuid.UUID) (*model.DocumentSummaryResponse, error)
	GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.DocumentSummaryListItem, int64, error)
	GetSummaryByID(ctx context.Context, id uuid.UUID) (*model.DocumentSummaryResponse, error)
	UpdateSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID, req model.UpdateDocumentSummaryRequest) (*model.DocumentSummaryResponse, error)
	DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ExportSummary(ctx context.Context, id uuid.UUID, format string) (*model.ExportFileResult, error)
}

type DocumentSummaryService struct {
	geminiClient gemini.IGeminiClient
	repo         repository.IDocumentSummaryRepository
}

func NewDocumentSummaryService(geminiClient gemini.IGeminiClient, repo repository.IDocumentSummaryRepository) *DocumentSummaryService {
	return &DocumentSummaryService{
		geminiClient: geminiClient,
		repo:         repo,
	}
}

func (s *DocumentSummaryService) SummarizeDocument(ctx context.Context, req model.SummarizeDocumentRequest, fileBytes []byte, filename string, contentType string, userID *uuid.UUID) (*model.DocumentSummaryResponse, error) {
	if len(fileBytes) == 0 {
		return nil, fmt.Errorf("uploaded file is empty")
	}

	// 1. Extract content / identify format
	extracted, err := document.ExtractTextFromDocument(filename, fileBytes, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to process document: %w", err)
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

	// 2. Prepare Gemini prompt and payload
	geminiReq := gemini.GeminiSummaryRequest{
		FileName:       filename,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
	}

	if extracted.FileType == "pdf" {
		geminiReq.InlineData = fileBytes
		geminiReq.MIMEType = "application/pdf"
	} else {
		geminiReq.TextContent = extracted.RawText
	}

	// 3. Call Gemini AI
	result, err := s.geminiClient.SummarizeContent(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("ai summarization error: %w", err)
	}

	// 4. Build response and persist to DB
	keyPointsJSON, _ := json.Marshal(result.KeyPoints)

	summaryID := uuid.New()
	summaryEntity := &entity.DocumentSummary{
		ID:             summaryID,
		UserID:         userID,
		FileName:       filename,
		FileType:       extracted.FileType,
		FileSize:       int64(len(fileBytes)),
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

	shouldSave := true
	if req.SaveToHistory != nil {
		shouldSave = *req.SaveToHistory
	}

	if shouldSave && s.repo != nil {
		if err := s.repo.CreateDocumentSummary(ctx, summaryEntity); err != nil {
			return nil, fmt.Errorf("failed to persist document summary to database: %w", err)
		}
	}

	return &model.DocumentSummaryResponse{
		ID:             summaryID,
		UserID:         userID,
		FileName:       filename,
		FileType:       extracted.FileType,
		FileSize:       int64(len(fileBytes)),
		Title:          result.Title,
		Summary:        result.Summary,
		KeyPoints:      result.KeyPoints,
		Explanation:    result.Explanation,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
		WordCount:      extracted.WordCount,
		SlideCount:     extracted.SlideCount,
		Model:          result.Model,
		TokenCount:     result.TokenCount,
		CreatedAt:      summaryEntity.CreatedAt,
	}, nil
}

func (s *DocumentSummaryService) SummarizeText(ctx context.Context, req model.SummarizeTextRequest, userID *uuid.UUID) (*model.DocumentSummaryResponse, error) {
	trimmed := strings.TrimSpace(req.Text)
	if trimmed == "" {
		return nil, fmt.Errorf("text cannot be empty")
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

	fileName := req.Title
	if fileName == "" {
		fileName = "Text Notes"
	}

	geminiReq := gemini.GeminiSummaryRequest{
		TextContent:    trimmed,
		FileName:       fileName,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
	}

	result, err := s.geminiClient.SummarizeContent(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("ai summarization error: %w", err)
	}

	keyPointsJSON, _ := json.Marshal(result.KeyPoints)

	summaryID := uuid.New()
	summaryEntity := &entity.DocumentSummary{
		ID:             summaryID,
		UserID:         userID,
		FileName:       fileName,
		FileType:       "text",
		FileSize:       int64(len(trimmed)),
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

	shouldSave := true
	if req.SaveToHistory != nil {
		shouldSave = *req.SaveToHistory
	}

	if shouldSave && s.repo != nil {
		if err := s.repo.CreateDocumentSummary(ctx, summaryEntity); err != nil {
			return nil, fmt.Errorf("failed to persist text summary to database: %w", err)
		}
	}

	return &model.DocumentSummaryResponse{
		ID:             summaryID,
		UserID:         userID,
		FileName:       fileName,
		FileType:       "text",
		FileSize:       int64(len(trimmed)),
		Title:          result.Title,
		Summary:        result.Summary,
		KeyPoints:      result.KeyPoints,
		Explanation:    result.Explanation,
		Language:       lang,
		DetailLevel:    detailLevel,
		TargetAudience: targetAudience,
		WordCount:      len(strings.Fields(trimmed)),
		SlideCount:     0,
		Model:          result.Model,
		TokenCount:     result.TokenCount,
		CreatedAt:      summaryEntity.CreatedAt,
	}, nil
}

func (s *DocumentSummaryService) GetSummariesByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.DocumentSummaryListItem, int64, error) {
	pagination.Check()
	entities, total, err := s.repo.GetSummariesByUserID(ctx, userID, pagination)
	if err != nil {
		return nil, 0, err
	}

	items := make([]model.DocumentSummaryListItem, len(entities))
	for i, e := range entities {
		preview := e.Summary
		if len(preview) > 160 {
			preview = preview[:157] + "..."
		}

		items[i] = model.DocumentSummaryListItem{
			ID:             e.ID,
			UserID:         e.UserID,
			FileName:       e.FileName,
			FileType:       e.FileType,
			FileSize:       e.FileSize,
			Title:          e.Title,
			SummaryPreview: preview,
			Language:       e.Language,
			CreatedAt:      e.CreatedAt,
		}
	}

	return items, total, nil
}

func (s *DocumentSummaryService) GetSummaryByID(ctx context.Context, id uuid.UUID) (*model.DocumentSummaryResponse, error) {
	summaryEntity, err := s.repo.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if summaryEntity == nil {
		return nil, nil
	}

	var keyPoints []string
	if summaryEntity.KeyPoints != "" {
		_ = json.Unmarshal([]byte(summaryEntity.KeyPoints), &keyPoints)
	}

	return &model.DocumentSummaryResponse{
		ID:             summaryEntity.ID,
		UserID:         summaryEntity.UserID,
		FileName:       summaryEntity.FileName,
		FileType:       summaryEntity.FileType,
		FileSize:       summaryEntity.FileSize,
		Title:          summaryEntity.Title,
		Summary:        summaryEntity.Summary,
		KeyPoints:      keyPoints,
		Explanation:    summaryEntity.Explanation,
		Language:       summaryEntity.Language,
		DetailLevel:    summaryEntity.DetailLevel,
		TargetAudience: summaryEntity.TargetAudience,
		TokenCount:     summaryEntity.TokenCount,
		CreatedAt:      summaryEntity.CreatedAt,
	}, nil
}

func (s *DocumentSummaryService) UpdateSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID, req model.UpdateDocumentSummaryRequest) (*model.DocumentSummaryResponse, error) {
	summaryEntity, err := s.repo.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if summaryEntity == nil {
		return nil, fmt.Errorf("document summary not found")
	}

	// Verify ownership
	if summaryEntity.UserID == nil || *summaryEntity.UserID != userID {
		return nil, fmt.Errorf("unauthorized to update this document summary")
	}

	if req.Title != nil {
		summaryEntity.Title = strings.TrimSpace(*req.Title)
	}
	if req.Summary != nil {
		summaryEntity.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.KeyPoints != nil {
		keyPointsJSON, _ := json.Marshal(*req.KeyPoints)
		summaryEntity.KeyPoints = string(keyPointsJSON)
	}
	if req.Explanation != nil {
		summaryEntity.Explanation = strings.TrimSpace(*req.Explanation)
	}
	if req.Language != nil {
		summaryEntity.Language = strings.TrimSpace(*req.Language)
	}
	if req.DetailLevel != nil {
		summaryEntity.DetailLevel = strings.TrimSpace(*req.DetailLevel)
	}
	if req.TargetAudience != nil {
		summaryEntity.TargetAudience = strings.TrimSpace(*req.TargetAudience)
	}

	summaryEntity.UpdatedAt = time.Now()

	if err := s.repo.UpdateDocumentSummary(ctx, summaryEntity); err != nil {
		return nil, fmt.Errorf("failed to update document summary: %w", err)
	}

	var keyPoints []string
	if summaryEntity.KeyPoints != "" {
		_ = json.Unmarshal([]byte(summaryEntity.KeyPoints), &keyPoints)
	}

	return &model.DocumentSummaryResponse{
		ID:             summaryEntity.ID,
		UserID:         summaryEntity.UserID,
		FileName:       summaryEntity.FileName,
		FileType:       summaryEntity.FileType,
		FileSize:       summaryEntity.FileSize,
		Title:          summaryEntity.Title,
		Summary:        summaryEntity.Summary,
		KeyPoints:      keyPoints,
		Explanation:    summaryEntity.Explanation,
		Language:       summaryEntity.Language,
		DetailLevel:    summaryEntity.DetailLevel,
		TargetAudience: summaryEntity.TargetAudience,
		TokenCount:     summaryEntity.TokenCount,
		CreatedAt:      summaryEntity.CreatedAt,
	}, nil
}

func (s *DocumentSummaryService) DeleteSummary(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeleteSummary(ctx, id, userID)
}

func (s *DocumentSummaryService) ExportSummary(ctx context.Context, id uuid.UUID, format string) (*model.ExportFileResult, error) {
	summaryEntity, err := s.repo.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve document summary: %w", err)
	}
	if summaryEntity == nil {
		return nil, fmt.Errorf("document summary not found")
	}

	var keyPoints []string
	if summaryEntity.KeyPoints != "" {
		_ = json.Unmarshal([]byte(summaryEntity.KeyPoints), &keyPoints)
	}

	sourceType := "Document (" + strings.ToUpper(summaryEntity.FileType) + ")"
	if summaryEntity.FileType == "speech" || summaryEntity.FileType == "stt" {
		sourceType = "Speech-to-Text Transcript"
	} else if summaryEntity.FileType == "audio" {
		sourceType = "Audio Recording"
	}

	payload := export.ExportPayload{
		Title:          summaryEntity.Title,
		Summary:        summaryEntity.Summary,
		KeyPoints:      keyPoints,
		Explanation:    summaryEntity.Explanation,
		Language:       summaryEntity.Language,
		DetailLevel:    summaryEntity.DetailLevel,
		TargetAudience: summaryEntity.TargetAudience,
		SourceType:     sourceType,
		CreatedAt:      summaryEntity.CreatedAt,
	}

	normFormat := strings.ToLower(strings.TrimSpace(format))
	baseName := strings.ReplaceAll(summaryEntity.Title, " ", "_")
	if baseName == "" {
		baseName = "summary"
	}
	// Sanitize filename
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
