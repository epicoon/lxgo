package models

import (
	"math/rand"

	"gorm.io/gorm"
)

type Client struct {
	gorm.Model
	Secret               string `gorm:"not null"`
	RoleID               uint   `gorm:"not null"`
	AccessTokenLifetime  uint   `gorm:"not null"`
	RefreshTokenLifetime uint   `gorm:"not null"`
	Role                 Role
	Tokens               []Token `gorm:"foreignKey:ClientID;"`
}

func (c *Client) GenSecret(n int) string {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}
