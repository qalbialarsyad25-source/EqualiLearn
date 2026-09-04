package delivery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"EquiliLearn/internal/model"
	"EquiliLearn/internal/service"
	"EquiliLearn/pkg/jwt"

	"net/http"

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
	defer session.Close()

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

	// 5. Goroutine for receiving recognition results and sending to client
	go func() {
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

				// Push transcript update to client
				event := model.TranscriptEvent{
					SessionID:  sessionID,
					Text:       result.Text,
					IsFinal:    result.IsFinal,
					Confidence: result.Confidence,
					Language:   config.LanguageCode,
					Timestamp:  time.Now(),
				}

				_ = safeWriteJSON(model.WSGenericMessage{
					Type:    model.WSMsgTypeTranscript,
					Payload: event,
				})

				// If the transcript sentence is finalized, persist to database
				if result.IsFinal && result.Text != "" {
					durationMs := time.Since(startTime).Milliseconds()
					_, _ = h.speechService.SaveFinalTranscription(
						context.Background(),
						userUUID,
						sessionID,
						config.LanguageCode,
						result.Text,
						result.Confidence,
						durationMs,
					)
				}
			}
		}
	}()

	// 6. Main loop: Read incoming audio chunks or config frames from WebSocket
	for {
		msgType, message, err := conn.ReadMessage()
		if err != nil {
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

	// Session finished
	_ = safeWriteJSON(model.WSGenericMessage{
		Type: model.WSMsgTypeFinished,
		Payload: gin.H{
			"session_id":  sessionID,
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	})
}
