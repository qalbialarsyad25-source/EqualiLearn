package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"EquiliLearn/internal/model"
	"EquiliLearn/internal/service"
	"EquiliLearn/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatWSHandler struct {
	hub         *ChatHub
	chatService service.IGroupChatService
	jwtService  jwt.IJWT
	upgrader    websocket.Upgrader
}

func NewChatWSHandler(hub *ChatHub, chatService service.IGroupChatService, jwtService jwt.IJWT) *ChatWSHandler {
	return &ChatWSHandler{
		hub:         hub,
		chatService: chatService,
		jwtService:  jwtService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow cross-origin WebSocket connections
			},
		},
	}
}

func (h *ChatWSHandler) HandleGroupChat(c *gin.Context) {
	// 1. Authenticate user from query parameter (?token=...) or Authorization header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication token"})
		return
	}

	userIDStr, _, err := h.jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user identity in token"})
		return
	}

	// 2. Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("[ChatWS] Failed to upgrade websocket connection: %v\n", err)
		return
	}
	defer conn.Close()

	// 3. Register client in Hub
	h.hub.AddClient(userIDStr, conn)
	defer h.hub.RemoveClient(userIDStr, conn)

	// Send initial connection ack
	_ = conn.WriteJSON(model.WSOutgoingMessage{
		Type: model.WSChatMsgTypeConnected,
		Payload: map[string]interface{}{
			"user_id": userIDStr,
			"status":  "authenticated",
		},
		Timestamp: time.Now(),
	})

	// Optional: auto-join active group passed in query
	initialGroupID := c.Query("group_id")
	if initialGroupID != "" {
		if gUUID, parseErr := uuid.Parse(initialGroupID); parseErr == nil {
			isMember, _, _ := h.chatService.ValidateGroupMembership(context.Background(), gUUID, userID)
			if isMember {
				h.hub.SubscribeGroup(userIDStr, initialGroupID)
			}
		}
	}

	// 4. Message pump loop
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("[ChatWS] WebSocket closed unexpectedly: %v\n", err)
			}
			break
		}

		var incoming model.WSIncomingMessage
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			_ = conn.WriteJSON(model.WSOutgoingMessage{
				Type:      model.WSChatMsgTypeError,
				Payload:   "invalid json payload",
				Timestamp: time.Now(),
			})
			continue
		}

		h.handleIncomingMessage(c.Request.Context(), userID, userIDStr, conn, incoming)
	}
}

func (h *ChatWSHandler) handleIncomingMessage(ctx context.Context, userID uuid.UUID, userIDStr string, conn *websocket.Conn, incoming model.WSIncomingMessage) {
	switch incoming.Type {
	case model.WSChatMsgTypeJoinGroup:
		groupID, err := uuid.Parse(incoming.GroupID)
		if err != nil {
			_ = conn.WriteJSON(model.WSOutgoingMessage{
				Type:      model.WSChatMsgTypeError,
				Payload:   "invalid group_id uuid",
				Timestamp: time.Now(),
			})
			return
		}

		isMember, _, err := h.chatService.ValidateGroupMembership(ctx, groupID, userID)
		if err != nil || !isMember {
			_ = conn.WriteJSON(model.WSOutgoingMessage{
				Type:      model.WSChatMsgTypeError,
				GroupID:   incoming.GroupID,
				Payload:   "access denied: you are not a member of this group",
				Timestamp: time.Now(),
			})
			return
		}

		h.hub.SubscribeGroup(userIDStr, incoming.GroupID)

		// Broadcast join notification to room
		h.hub.BroadcastToGroup(incoming.GroupID, model.WSOutgoingMessage{
			Type:    model.WSChatMsgTypeJoinGroup,
			GroupID: incoming.GroupID,
			Payload: map[string]interface{}{
				"user_id": userIDStr,
				"action":  "joined_room",
			},
			Timestamp: time.Now(),
		})

	case model.WSChatMsgTypeLeaveGroup:
		h.hub.UnsubscribeGroup(userIDStr, incoming.GroupID)
		h.hub.BroadcastToGroup(incoming.GroupID, model.WSOutgoingMessage{
			Type:    model.WSChatMsgTypeLeaveGroup,
			GroupID: incoming.GroupID,
			Payload: map[string]interface{}{
				"user_id": userIDStr,
				"action":  "left_room",
			},
			Timestamp: time.Now(),
		})

	case model.WSChatMsgTypeMessage:
		groupID, err := uuid.Parse(incoming.GroupID)
		if err != nil {
			_ = conn.WriteJSON(model.WSOutgoingMessage{
				Type:      model.WSChatMsgTypeError,
				Payload:   "invalid group_id uuid",
				Timestamp: time.Now(),
			})
			return
		}

		content := strings.TrimSpace(incoming.Content)
		if content == "" {
			return
		}

		// Ensure user is subscribed in hub
		h.hub.SubscribeGroup(userIDStr, incoming.GroupID)

		savedMsg, err := h.chatService.SaveMessage(ctx, groupID, userID, content, incoming.MessageType)
		if err != nil {
			_ = conn.WriteJSON(model.WSOutgoingMessage{
				Type:      model.WSChatMsgTypeError,
				GroupID:   incoming.GroupID,
				Payload:   fmt.Sprintf("failed to save message: %v", err),
				Timestamp: time.Now(),
			})
			return
		}

		// Broadcast message to everyone in the group room
		h.hub.BroadcastToGroup(incoming.GroupID, model.WSOutgoingMessage{
			Type:      model.WSChatMsgTypeMessage,
			GroupID:   incoming.GroupID,
			Payload:   savedMsg,
			Timestamp: savedMsg.CreatedAt,
		})

	case model.WSChatMsgTypeTyping:
		if incoming.GroupID == "" {
			return
		}
		// Broadcast typing status to everyone currently viewing the group room
		h.hub.BroadcastToGroup(incoming.GroupID, model.WSOutgoingMessage{
			Type:    model.WSChatMsgTypeTyping,
			GroupID: incoming.GroupID,
			Payload: map[string]interface{}{
				"user_id":  userIDStr,
				"group_id": incoming.GroupID,
			},
			Timestamp: time.Now(),
		})
	}
}

// Hub returns the underlying ChatHub instance for external notifications.
func (h *ChatWSHandler) Hub() *ChatHub {
	return h.hub
}
