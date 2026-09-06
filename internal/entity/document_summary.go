package entity

import (
	"time"

	"github.com/google/uuid"
)

// DocumentSummary represents a stored AI-generated summary and explanation of an uploaded document or text.
type DocumentSummary struct {
	ID             uuid.UUID  `gorm:"primaryKey;type:uuid" json:"id"`
	UserID         *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	FileName       string     `gorm:"type:varchar(255);not null" json:"file_name"`
	FileType       string     `gorm:"type:varchar(50);not null" json:"file_type"` // "pdf", "pptx", "txt", "text"
	FileSize       int64      `gorm:"type:bigint;default:0" json:"file_size"`
	Title          string     `gorm:"type:varchar(255)" json:"title"`
	Summary        string     `gorm:"type:text;not null" json:"summary"`
	KeyPoints      string     `gorm:"type:text" json:"key_points"`       // JSON string or newline-separated key points
	Explanation    string     `gorm:"type:text" json:"explanation"`     // In-depth conceptual explanation
	Language       string     `gorm:"type:varchar(20);default:'id'" json:"language"`
	DetailLevel    string     `gorm:"type:varchar(50);default:'balanced'" json:"detail_level"`
	TargetAudience string     `gorm:"type:varchar(50);default:'general'" json:"target_audience"`
	TokenCount     int        `gorm:"type:int;default:0" json:"token_count,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}
