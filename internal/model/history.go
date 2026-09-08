package model

import (
	"time"

	"github.com/google/uuid"
)

// HistoryItemType defines constants for history record types.
const (
	HistoryTypeDocumentSummary = "document_summary"
	HistoryTypeSTT             = "stt"
	HistoryTypeTTS             = "tts"
	HistoryTypeAll             = "all"
)

// UnifiedHistoryItem represents a single timeline item combining Document Summary, STT, or TTS.
type UnifiedHistoryItem struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"` // "document_summary", "stt", "tts"
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	Language    string    `json:"language,omitempty"`
	DurationMs  int64     `json:"duration_ms,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Details     any       `json:"details,omitempty"` // DocumentSummaryResponse, TranscriptionResponse, or TTSHistoryResponse
}

// UnifiedHistoryQuery defines parameters for retrieving unified history items.
type UnifiedHistoryQuery struct {
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
	Type   string `form:"type"`   // "all", "document_summary" / "summary", "stt", "tts", or comma-separated
	Search string `form:"search"` // Keyword search in text / title / file_name
}

// HistoryCounts contains aggregate counts across all activity types for a user.
type HistoryCounts struct {
	TotalSummaries int64 `json:"total_summaries"`
	TotalSTT       int64 `json:"total_stt"`
	TotalTTS       int64 `json:"total_tts"`
	TotalAll       int64 `json:"total_all"`
}

// UnifiedHistoryResponse represents the paginated unified history response.
type UnifiedHistoryResponse struct {
	Data       []UnifiedHistoryItem `json:"data"`
	Total      int64                `json:"total"`
	Counts     HistoryCounts        `json:"counts"`
	Pagination Pagination           `json:"pagination"`
}

// HistoryStatsResponse represents statistics of user's activities.
type HistoryStatsResponse struct {
	Counts              HistoryCounts `json:"counts"`
	TotalSpeechDuration int64         `json:"total_speech_duration_ms"`
	TotalTTSDuration    int64         `json:"total_tts_duration_ms"`
}
