package models

import (
	"gorm.io/gorm"
)

type UserData struct {
	gorm.Model
	UserID   uint  `gorm:"not null"`
	ClientID uint  `gorm:"not null"`
	Data     JSONB `gorm:"type:jsonb"`
}
