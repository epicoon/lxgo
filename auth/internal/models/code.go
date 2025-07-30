package models

type Code struct {
	ID       uint   `gorm:"primarykey"`
	Value    string `gorm:"not null"`
	ClientID uint   `gorm:"not null"`
	UserID   uint   `gorm:"not null"`
}
