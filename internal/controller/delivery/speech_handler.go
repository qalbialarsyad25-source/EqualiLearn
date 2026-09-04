package delivery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"EquiliLearn/internal/model"
	"EquiliLearn/internal/service"
	"EquiliLearn/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow CORS origins
	},
}

type SpeechWSHandler struct {
	manager       *WSManager
	speechService service.ISpeechService
	jwtService    jwt.IJWT
}

func NewSpeechWSHandler(manager *WSManager, speechService service.ISpeechService, jwtService jwt.IJWT) *SpeechWSHandler {
	return &SpeechWSHandler{
		manager:       manager,
		speechService: speechService,
		jwtService:    jwtService,
	}
}

// HandleRealtimeSTT handles WebSocket connection for real-time speech recognition
func (h *SpeechWSHandler) HandleRealtimeSTT(c *gin.Context) {
	// 1. Authenticate user from query param `token` or Authorization header (optional for guest demo)
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	var userUUID *uuid.UUID
	clientTrackingID := uuid.New().String()

	if tokenStr != "" {
		userIdStr, _, err := h.jwtService.ValidateToken(tokenStr)
		if err == nil {
			parsedID, err := uuid.Parse(userIdStr)
			if err == nil {
				userUUID = &parsedID
				clientTrackingID = parsedID.String()
			}
		}
	}

	// 2. Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.manager.AddClient(clientTrackingID, conn)
	defer h.manager.RemoveClient(clientTrackingID, conn)

	// 3. Prepare stream configuration defaults (can be overridden via query params)
	lang := c.DefaultQuery("lang", "id-ID")
	config := model.STTStreamConfig{
		LanguageCode:   lang,
		SampleRate:     16000,
		Encoding:       "linear16",
		InterimResults: true,
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 4. Start speech recognition session via SpeechService
	session, sessionID, err := h.speechService.StartSTTSession(ctx, userUUID, config)
	if err != nil {
		_ = conn.WriteJSON(model.WSGenericMessage{
			Type: model.WSMsgTypeError,
			Payload: gin.H{
				"message": "failed to start speech session: " + err.Error(),
			},
		})
		return
	}

	// Notify client that connection is ready to receive audio
	_ = conn.WriteJSON(model.WSGenericMessage{
		Type: model.WSMsgTypeReady,
		Payload: gin.H{
			"session_id":  sessionID,
			"language":    config.LanguageCode,
			"sample_rate": config.SampleRate,
		},
	})

	var writeMu sync.Mutex
	safeWriteJSON := func(v interface{}) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(v)
	}

	resultsChan, errorsChan := session.Receive()
	startTime := time.Now()

	// Track accumulated full transcript across the entire audio session
	var (
		accumulatedFinalTexts []string
		latestInterimText     string
		totalConfidence       float64
		confidenceCount       int
		textMu                sync.Mutex
		receiverWg            sync.WaitGroup
	)

	receiverWg.Add(1)

	// 5. Goroutine for receiving recognition results and pushing live to client
	go func() {
		defer receiverWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errorsChan:
				if !ok {
					return
				}
				if err != nil {
					_ = safeWriteJSON(model.WSGenericMessage{
						Type: model.WSMsgTypeError,
						Payload: gin.H{
							"message": err.Error(),
						},
					})
				}
			case result, ok := <-resultsChan:
				if !ok {
					return
				}

				trimmed := strings.TrimSpace(result.Text)

				// Push live transcript update to client
				event := model.TranscriptEvent{
					SessionID:  sessionID,
					Text:       trimmed,
					IsFinal:    result.IsFinal,
					Confidence: result.Confidence,
					Language:   config.LanguageCode,
					Timestamp:  time.Now(),
				}

				_ = safeWriteJSON(model.WSGenericMessage{
					Type:    model.WSMsgTypeTranscript,
					Payload: event,
				})

				// Accumulate text in memory during live session (DO NOT write to DB yet)
				textMu.Lock()
				if result.IsFinal {
					if trimmed != "" {
						accumulatedFinalTexts = append(accumulatedFinalTexts, trimmed)
						if result.Confidence > 0 {
							totalConfidence += result.Confidence
							confidenceCount++
						}
					}
					latestInterimText = ""
				} else {
					latestInterimText = trimmed
				}
				textMu.Unlock()
			}
		}
	}()

	// 6. Main loop: Read incoming audio chunks or config frames from WebSocket until disconnect
	for {
		msgType, message, err := conn.ReadMessage()
		if err != nil {
			// Client stopped audio stream / disconnected
			break
		}

		switch msgType {
		case websocket.BinaryMessage:
			// Raw audio bytes (PCM 16-bit 16kHz mono, etc.)
			if err := session.SendAudio(message); err != nil {
				break
			}

		case websocket.TextMessage:
			// JSON control messages (e.g. config update or base64 audio chunk)
			var genericMsg model.WSGenericMessage
			if err := json.Unmarshal(message, &genericMsg); err == nil {
				switch genericMsg.Type {
				case model.WSMsgTypeAudioChunk:
					if b64, ok := genericMsg.Payload.(string); ok {
						if audioBytes, err := base64.StdEncoding.DecodeString(b64); err == nil {
							_ = session.SendAudio(audioBytes)
						}
					}
				case model.WSMsgTypeConfig:
					// Dynamic config update
					if payloadBytes, err := json.Marshal(genericMsg.Payload); err == nil {
						var newCfg model.STTStreamConfig
						if err := json.Unmarshal(payloadBytes, &newCfg); err == nil {
							if newCfg.LanguageCode != "" {
								config.LanguageCode = newCfg.LanguageCode
							}
						}
					}
				}
			}
		}
	}

	// 7. Client disconnected / stopped audio: Close STT session and wait for receiver to finish
	_ = session.Close()
	cancel()
	receiverWg.Wait()

	// 8. Aggregate the full session text and save to DB ONCE after disconnection
	textMu.Lock()
	if latestInterimText != "" && len(accumulatedFinalTexts) == 0 {
		// In case user disconnected before a sentence finalized, preserve what was spoken
		accumulatedFinalTexts = append(accumulatedFinalTexts, latestInterimText)
	}

	fullTranscript := strings.TrimSpace(strings.Join(accumulatedFinalTexts, " "))
	avgConfidence := 0.95
	if confidenceCount > 0 {
		avgConfidence = totalConfidence / float64(confidenceCount)
	}
	textMu.Unlock()

	durationMs := time.Since(startTime).Milliseconds()

	if fullTranscript != "" {
		_, _ = h.speechService.SaveFinalTranscription(
			context.Background(),
			userUUID,
			sessionID,
			config.LanguageCode,
			fullTranscript,
			avgConfidence,
			durationMs,
		)
	}

	// Notify finished event if connection is still responsive
	_ = safeWriteJSON(model.WSGenericMessage{
		Type: model.WSMsgTypeFinished,
		Payload: gin.H{
			"session_id":  sessionID,
			"duration_ms": durationMs,
			"text":        fullTranscript,
		},
	})
}
