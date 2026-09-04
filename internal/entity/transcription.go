package entity

import (
	"time"

	"github.com/google/uuid"
)

type Transcription struct {
	ID         uuid.UUID  `gorm:"primaryKey;type:uuid" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	SessionID  string     `gorm:"type:varchar(100);index" json:"session_id"`
	Language   string     `gorm:"type:varchar(20);default:'id-ID'" json:"language"`
	Text       string     `gorm:"type:text;not null" json:"text"`
	Confidence float64    `gorm:"type:numeric(5,4)" json:"confidence"`
	DurationMs int64      `gorm:"type:bigint;default:0" json:"duration_ms"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}
