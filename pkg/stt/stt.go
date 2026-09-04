package stt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// STTConfig defines configuration for a streaming speech recognition session.
type STTConfig struct {
	LanguageCode   string `json:"language_code"`   // e.g. "id-ID", "en-US"
	SampleRate     int    `json:"sample_rate"`     // e.g. 16000
	Encoding       string `json:"encoding"`        // e.g. "linear16", "opus", "webm"
	InterimResults bool   `json:"interim_results"` // whether to emit partial transcripts
}

// TranscriptResult represents real-time speech-to-text output.
type TranscriptResult struct {
	Text         string  `json:"text"`
	IsFinal      bool    `json:"is_final"`
	Confidence   float64 `json:"confidence"`
	LanguageCode string  `json:"language_code,omitempty"`
}

// ISTTSession represents an active bidirectional streaming recognition session.
type ISTTSession interface {
	SendAudio(chunk []byte) error
	Receive() (<-chan TranscriptResult, <-chan error)
	Close() error
}

// ISTTClient is the factory interface for speech-to-text streaming providers.
type ISTTClient interface {
	StartStream(ctx context.Context, cfg STTConfig) (ISTTSession, error)
}

// ==========================================
// Deepgram Streaming STT Implementation
// ==========================================

type deepgramSession struct {
	conn       *websocket.Conn
	results    chan TranscriptResult
	errors     chan error
	closeOnce  sync.Once
	closed     chan struct{}
	cancelFunc context.CancelFunc
}

type deepgramResponse struct {
	Type    string `json:"type"`
	Channel struct {
		Alternatives []struct {
			Transcript string  `json:"transcript"`
			Confidence float64 `json:"confidence"`
		} `json:"alternatives"`
	} `json:"channel"`
	IsFinal  bool `json:"is_final"`
	SpeechFinal bool `json:"speech_final"`
}

type DeepgramClient struct {
	apiKey string
}

func NewDeepgramClient(apiKey string) *DeepgramClient {
	return &DeepgramClient{apiKey: apiKey}
}

func (c *DeepgramClient) StartStream(ctx context.Context, cfg STTConfig) (ISTTSession, error) {
	lang := cfg.LanguageCode
	if lang == "" {
		lang = "id" // default Indonesian or fallback
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	u := url.URL{
		Scheme: "wss",
		Host:   "api.deepgram.com",
		Path:   "/v1/listen",
	}

	q := u.Query()
	q.Set("language", lang)
	q.Set("sample_rate", fmt.Sprintf("%d", sampleRate))
	q.Set("interim_results", fmt.Sprintf("%t", cfg.InterimResults))
	q.Set("punctuate", "true")
	q.Set("smart_format", "true")
	if strings.ToLower(cfg.Encoding) == "linear16" || strings.ToLower(cfg.Encoding) == "pcm" {
		q.Set("encoding", "linear16")
	}
	u.RawQuery = q.Encode()

	headers := http.Header{}
	headers.Add("Authorization", "Token "+c.apiKey)

	streamCtx, cancel := context.WithCancel(ctx)
	conn, _, err := websocket.DefaultDialer.DialContext(streamCtx, u.String(), headers)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to Deepgram STT stream: %w", err)
	}

	session := &deepgramSession{
		conn:       conn,
		results:    make(chan TranscriptResult, 64),
		errors:     make(chan error, 16),
		closed:     make(chan struct{}),
		cancelFunc: cancel,
	}

	go session.readLoop()

	return session, nil
}

func (s *deepgramSession) readLoop() {
	defer close(s.results)
	defer close(s.errors)

	for {
		select {
		case <-s.closed:
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					select {
					case s.errors <- err:
					default:
					}
				}
				return
			}

			var resp deepgramResponse
			if err := json.Unmarshal(message, &resp); err != nil {
				continue
			}

			if len(resp.Channel.Alternatives) > 0 {
				alt := resp.Channel.Alternatives[0]
				if strings.TrimSpace(alt.Transcript) != "" {
					s.results <- TranscriptResult{
						Text:       alt.Transcript,
						IsFinal:    resp.IsFinal || resp.SpeechFinal,
						Confidence: alt.Confidence,
					}
				}
			}
		}
	}
}

func (s *deepgramSession) SendAudio(chunk []byte) error {
	select {
	case <-s.closed:
		return fmt.Errorf("session is closed")
	default:
		return s.conn.WriteMessage(websocket.BinaryMessage, chunk)
	}
}

func (s *deepgramSession) Receive() (<-chan TranscriptResult, <-chan error) {
	return s.results, s.errors
}

func (s *deepgramSession) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		s.cancelFunc()
		_ = s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"))
		_ = s.conn.Close()
	})
	return nil
}

// ==========================================
// Mock / Simulation STT Implementation (Local Development)
// ==========================================

type mockSession struct {
	cfg        STTConfig
	results    chan TranscriptResult
	errors     chan error
	closeOnce  sync.Once
	closed     chan struct{}
	audioCount int
	mu         sync.Mutex
}

type MockSTTClient struct{}

func NewMockSTTClient() *MockSTTClient {
	return &MockSTTClient{}
}

func (m *MockSTTClient) StartStream(ctx context.Context, cfg STTConfig) (ISTTSession, error) {
	session := &mockSession{
		cfg:     cfg,
		results: make(chan TranscriptResult, 32),
		errors:  make(chan error, 8),
		closed:  make(chan struct{}),
	}
	return session, nil
}

func (s *mockSession) SendAudio(chunk []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.closed:
		return fmt.Errorf("mock session is closed")
	default:
	}

	if len(chunk) == 0 {
		return nil
	}

	s.audioCount++

	// Simulate real-time streaming recognition feedback every few audio packets received
	if s.audioCount%5 == 0 {
		samplePhrasesID := []string{
			"Halo",
			"Halo selamat datang",
			"Halo selamat datang di platform",
			"Halo selamat datang di platform EquiliLearn",
			"EquiliLearn mendukung pembelajaran inklusif dan berkualitas.",
		}
		samplePhrasesEN := []string{
			"Hello",
			"Hello welcome",
			"Hello welcome to EquiliLearn",
			"EquiliLearn platform supports inclusive education for all.",
		}

		phrases := samplePhrasesID
		if strings.HasPrefix(strings.ToLower(s.cfg.LanguageCode), "en") {
			phrases = samplePhrasesEN
		}

		idx := (s.audioCount / 5) - 1
		if idx < len(phrases) {
			isFinal := idx == len(phrases)-1
			select {
			case s.results <- TranscriptResult{
				Text:         phrases[idx],
				IsFinal:      isFinal,
				Confidence:   0.94 + float64(idx)*0.01,
				LanguageCode: s.cfg.LanguageCode,
			}:
			case <-time.After(20 * time.Millisecond):
			}
		}
	}

	return nil
}

func (s *mockSession) Receive() (<-chan TranscriptResult, <-chan error) {
	return s.results, s.errors
}

func (s *mockSession) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
		close(s.results)
		close(s.errors)
	})
	return nil
}

// ==========================================
// Factory Provider Initializer
// ==========================================

func NewSTTClient() ISTTClient {
	provider := strings.ToLower(os.Getenv("STT_PROVIDER"))
	deepgramKey := os.Getenv("DEEPGRAM_API_KEY")

	if provider == "deepgram" && deepgramKey != "" {
		return NewDeepgramClient(deepgramKey)
	}

	// Default fallback to mock client if no external API key is provided
	return NewMockSTTClient()
}
