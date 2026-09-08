package entity

import (
	"time"

	"github.com/google/uuid"
)

// TTSHistory represents a recorded text-to-speech audio synthesis generation.
type TTSHistory struct {
	ID         uuid.UUID  `gorm:"primaryKey;type:uuid" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Text       string     `gorm:"type:text;not null" json:"text"`
	Voice      string     `gorm:"type:varchar(100);not null" json:"voice"`
	Format     string     `gorm:"type:varchar(20);default:'mp3'" json:"format"`
	DurationMs int64      `gorm:"type:bigint;default:0" json:"duration_ms"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}
