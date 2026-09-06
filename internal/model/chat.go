package model

import (
	"time"

	"github.com/google/uuid"
)

// UserSearchResponse represents summary info returned when searching for users by email or name.
type UserSearchResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// CreateGroupRequest represents the payload to create a new group chat.
type CreateGroupRequest struct {
	Name          string      `json:"name" validate:"required"`
	Description   string      `json:"description"`
	MemberUserIDs []uuid.UUID `json:"member_user_ids,omitempty"`
	MemberEmails  []string    `json:"member_emails,omitempty"`
}

// AddGroupMemberRequest represents adding a new member to an existing group.
type AddGroupMemberRequest struct {
	UserID *uuid.UUID `json:"user_id,omitempty"`
	Email  string     `json:"email,omitempty"`
	Role   string     `json:"role,omitempty"` // "admin", "member" (default "member")
}

// GroupMemberResponse represents member details in a group.
type GroupMemberResponse struct {
	ID       uuid.UUID `json:"id"`
	GroupID  uuid.UUID `json:"group_id"`
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

// GroupResponse represents a summary of a group for list views.
type GroupResponse struct {
	ID             uuid.UUID             `json:"id"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	CreatedByID    uuid.UUID             `json:"created_by_id"`
	CreatedByName  string                `json:"created_by_name"`
	MemberCount    int                   `json:"member_count"`
	UserRole       string                `json:"user_role,omitempty"`
	LastMessage    *GroupMessageResponse `json:"last_message,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// GroupDetailResponse represents detailed group info including its member list.
type GroupDetailResponse struct {
	ID            uuid.UUID             `json:"id"`
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	CreatedByID   uuid.UUID             `json:"created_by_id"`
	CreatedByName string                `json:"created_by_name"`
	Members       []GroupMemberResponse `json:"members"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// GroupMessageResponse represents a chat message sent inside a group.
type GroupMessageResponse struct {
	ID          uuid.UUID `json:"id"`
	GroupID     uuid.UUID `json:"group_id"`
	SenderID    uuid.UUID `json:"sender_id"`
	SenderName  string    `json:"sender_name"`
	SenderEmail string    `json:"sender_email"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// WebSocket Message Protocol Constants
const (
	WSChatMsgTypeMessage     = "chat_message"
	WSChatMsgTypeTyping      = "typing"
	WSChatMsgTypeJoinGroup   = "join_group"
	WSChatMsgTypeLeaveGroup  = "leave_group"
	WSChatMsgTypeMemberAdded = "member_added"
	WSChatMsgTypeMemberLeft  = "member_left"
	WSChatMsgTypeError       = "error"
	WSChatMsgTypeConnected   = "connected"
)

// WSIncomingMessage represents message received from client WebSocket connection.
type WSIncomingMessage struct {
	Type        string `json:"type"`                  // "chat_message", "typing", "join_group", "leave_group"
	GroupID     string `json:"group_id"`              // Target group UUID
	Content     string `json:"content,omitempty"`     // Message text
	MessageType string `json:"message_type,omitempty"`// "text", "file", etc.
}

// WSOutgoingMessage represents message broadcast to client WebSocket connections.
type WSOutgoingMessage struct {
	Type      string      `json:"type"`
	GroupID   string      `json:"group_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// WSTypingPayload represents a typing indicator event.
type WSTypingPayload struct {
	GroupID    string    `json:"group_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	UserEmail  string    `json:"user_email"`
	IsTyping   bool      `json:"is_typing"`
}
