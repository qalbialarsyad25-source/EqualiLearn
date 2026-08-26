package entity

import (
	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID `gorm:"primaryKey;type:uuid"`
	Name     string    `gorm:"type:varchar(100);not null"`
	Email    string    `gorm:"type:varchar(100);not null;unique"`
	Password string    `gorm:"type:varchar(255);not null"`
}