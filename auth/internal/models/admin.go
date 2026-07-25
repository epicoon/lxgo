package models

import (
	"gorm.io/gorm"
)

// Admin marks a User as an operator of the auth service itself (as opposed
// to an end user of some Client application).
type Admin struct {
	gorm.Model
	UserID uint `gorm:"not null"`
	RoleID uint `gorm:"not null"`
	Role   Role
}
