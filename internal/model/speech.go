package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	WSMsgTypeConfig     = "config"     // Client -> Server: Configure STT parameters
	WSMsgTypeAudioChunk = "audio"      // Client -> Server: Audio chunk (if sending JSON base64)
	WSMsgTypeTranscript = "transcript" // Server -> Client: Real-time transcript update
	WSMsgTypeError      = "error"      // Server -> Client: Error notification
	WSMsgTypeReady      = "ready"      // Server -> Client: Stream is initialized and ready
	WSMsgTypeFinished   = "finished"   // Server -> Client: Session completed
	WSMsgTypeSummary    = "summary"    // Server -> Client: AI generated summary from speech
)

type WSGenericMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type STTStreamConfig struct {
	LanguageCode   string `json:"language_code" validate:"omitempty,oneof=id-ID en-US id en"`
	SampleRate     int    `json:"sample_rate"`
	Encoding       string `json:"encoding"`
	InterimResults bool   `json:"interim_results"`
}

type TranscriptEvent struct {
	SessionID  string    `json:"session_id,omitempty"`
	Text       string    `json:"text"`
	IsFinal    bool      `json:"is_final"`
	Confidence float64   `json:"confidence"`
	Language   string    `json:"language,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type TranscriptionResponse struct {
	ID         uuid.UUID  `json:"id"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	SessionID  string     `json:"session_id"`
	Language   string     `json:"language"`
	Text       string     `json:"text"`
	Confidence float64    `json:"confidence"`
	DurationMs int64      `json:"duration_ms"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SummarizeSpeechRequest defines parameters for generating AI summaries from speech text or an existing transcription ID.
type SummarizeSpeechRequest struct {
	TranscriptionID *uuid.UUID `json:"transcription_id,omitempty" form:"transcription_id"`
	Text            string     `json:"text,omitempty" form:"text"`
	Title           string     `json:"title,omitempty" form:"title"`
	Language        string     `json:"language,omitempty" form:"language"`
	DetailLevel     string     `json:"detail_level,omitempty" form:"detail_level"`
	TargetAudience  string     `json:"target_audience,omitempty" form:"target_audience"`
	SaveToHistory   *bool      `json:"save_to_history,omitempty" form:"save_to_history"`
}

// SummarizeSpeechAudioRequest defines query/form parameters for summarizing uploaded speech audio files.
type SummarizeSpeechAudioRequest struct {
	Title          string `form:"title"`
	Language       string `form:"language"`
	DetailLevel    string `form:"detail_level"`
	TargetAudience string `form:"target_audience"`
	SaveToHistory  *bool  `form:"save_to_history"`
}

// SpeechSummaryResponse represents the AI structured summary result generated from speech.
type SpeechSummaryResponse struct {
	ID              uuid.UUID  `json:"id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`
	TranscriptionID *uuid.UUID `json:"transcription_id,omitempty"`
	Title           string     `json:"title"`
	TranscriptText  string     `json:"transcript_text"`
	Summary         string     `json:"summary"`
	KeyPoints       []string   `json:"key_points"`
	Explanation     string     `json:"explanation"`
	Language        string     `json:"language"`
	DetailLevel     string     `json:"detail_level"`
	TargetAudience  string     `json:"target_audience"`
	Model           string     `json:"model,omitempty"`
	TokenCount      int        `json:"token_count,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
