package models

import (
	"gorm.io/gorm"
)

// TODO
// - Name, Phone, Email - need self auth_client to manage this fields
type User struct {
	gorm.Model
	Login    string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Name     string
	Phone    string
	Email    string
}
