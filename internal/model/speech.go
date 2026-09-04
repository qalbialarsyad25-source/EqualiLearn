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
