package entity

import (
	"time"

	"github.com/google/uuid"
)

// Group represents a collaborative chat group created by a user.
type Group struct {
	ID          uuid.UUID     `gorm:"primaryKey;type:uuid" json:"id"`
	Name        string        `gorm:"type:varchar(100);not null" json:"name"`
	Description string        `gorm:"type:text" json:"description"`
	CreatedByID uuid.UUID     `gorm:"type:uuid;not null;index" json:"created_by_id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	CreatedBy   *User         `gorm:"foreignKey:CreatedByID;constraint:OnDelete:CASCADE" json:"created_by,omitempty"`
	Members     []GroupMember `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"members,omitempty"`
	Messages    []GroupMessage`gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"-"`
}

// GroupMember represents membership of a user in a chat group.
type GroupMember struct {
	ID       uuid.UUID `gorm:"primaryKey;type:uuid" json:"id"`
	GroupID  uuid.UUID `gorm:"type:uuid;not null;index:idx_group_user,unique" json:"group_id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index:idx_group_user,unique" json:"user_id"`
	Role     string    `gorm:"type:varchar(20);default:'member'" json:"role"` // "admin", "member"
	JoinedAt time.Time `json:"joined_at"`

	Group    *Group    `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"group,omitempty"`
	User     *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// GroupMessage represents a persisted chat message sent within a group.
type GroupMessage struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid" json:"id"`
	GroupID     uuid.UUID `gorm:"type:uuid;not null;index" json:"group_id"`
	SenderID    uuid.UUID `gorm:"type:uuid;not null;index" json:"sender_id"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	MessageType string    `gorm:"type:varchar(20);default:'text'" json:"message_type"` // "text", "system", "file"
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Group       *Group    `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"group,omitempty"`
	Sender      *User     `gorm:"foreignKey:SenderID;constraint:OnDelete:CASCADE" json:"sender,omitempty"`
}
