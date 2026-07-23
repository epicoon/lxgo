package models

import (
	"math/rand"

	"gorm.io/gorm"
)

const (
	DefaultAccessTokenLifetime  = 900
	DefaultRefreshTokenLifetime = 604800
)

type Client struct {
	gorm.Model
	Secret               string  `gorm:"not null"`
	AccessTokenLifetime  uint    `gorm:"not null"`
	RefreshTokenLifetime uint    `gorm:"not null"`
	RedirectUri          string  `gorm:"not null"`
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
