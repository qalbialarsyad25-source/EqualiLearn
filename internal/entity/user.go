package entity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID  `gorm:"primaryKey;type:uuid" json:"id"`
	Name              string     `gorm:"type:varchar(100);not null" json:"name"`
	Email             string     `gorm:"type:varchar(100);not null;unique" json:"email"`
	Password          string     `gorm:"type:varchar(255);not null" json:"-"`
	Role              string     `gorm:"type:varchar(50);default:'user'" json:"role"`
	ResetToken        string     `gorm:"type:varchar(255)" json:"-"`
	ResetTokenExpired *time.Time `json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}