package model

import (
	"time"

	"github.com/google/uuid"
)

// SummarizeDocumentRequest represents multipart form parameters for uploading and summarizing documents.
type SummarizeDocumentRequest struct {
	Language       string `form:"language" json:"language"`               // "id", "en"
	DetailLevel    string `form:"detail_level" json:"detail_level"`       // "brief", "balanced", "detailed"
	TargetAudience string `form:"target_audience" json:"target_audience"` // "student", "beginner", "professional", "general"
	SaveToHistory  *bool  `form:"save_to_history" json:"save_to_history"`
}

// SummarizeTextRequest represents a JSON request to summarize raw text or lecture notes.
type SummarizeTextRequest struct {
	Title          string `json:"title"`
	Text           string `json:"text" validate:"required"`
	Language       string `json:"language"`
	DetailLevel    string `json:"detail_level"`
	TargetAudience string `json:"target_audience"`
	SaveToHistory  *bool  `json:"save_to_history"`
}

// DocumentSummaryResponse represents the comprehensive output of a document summarization.
type DocumentSummaryResponse struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	FileName       string     `json:"file_name"`
	FileType       string     `json:"file_type"`
	FileSize       int64      `json:"file_size"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	KeyPoints      []string   `json:"key_points"`
	Explanation    string     `json:"explanation"`
	Language       string     `json:"language"`
	DetailLevel    string     `json:"detail_level"`
	TargetAudience string     `json:"target_audience"`
	WordCount      int        `json:"word_count,omitempty"`
	SlideCount     int        `json:"slide_count,omitempty"`
	Model          string     `json:"model,omitempty"`
	TokenCount     int        `json:"token_count,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// DocumentSummaryListItem represents a concise summary item for history list views.
type DocumentSummaryListItem struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	FileName       string     `json:"file_name"`
	FileType       string     `json:"file_type"`
	FileSize       int64      `json:"file_size"`
	Title          string     `json:"title"`
	SummaryPreview string     `json:"summary_preview"`
	Language       string     `json:"language"`
	CreatedAt      time.Time  `json:"created_at"`
}

// UpdateDocumentSummaryRequest represents client payload to update an existing summary's title, summary, key points, explanation, etc.
type UpdateDocumentSummaryRequest struct {
	Title          *string   `json:"title"`
	Summary        *string   `json:"summary"`
	KeyPoints      *[]string `json:"key_points"`
	Explanation    *string   `json:"explanation"`
	Language       *string   `json:"language"`
	DetailLevel    *string   `json:"detail_level"`
	TargetAudience *string   `json:"target_audience"`
}

// ExportFileResult represents binary file payload (PDF or Text) with filename and content type headers.
type ExportFileResult struct {
	Data        []byte
	Filename    string
	ContentType string
}

