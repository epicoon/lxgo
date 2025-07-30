package models

import (
	"time"

	"github.com/epicoon/lxgo/kernel/utils"
)

type Token struct {
	ID        uint      `gorm:"primaryKey"`
	ClientID  uint      `gorm:"not null"`
	UserID    uint      `gorm:"not null"`
	Value     string    `gorm:"not null"`
	IsRefresh bool      `gorm:"not null"`
	ExpiredAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (t *Token) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiredAt)
}

func (t *Token) Refresh(client *Client) {
	t.Value = utils.GenRandomHash(16)
	if t.IsRefresh {
		t.ExpiredAt = time.Now().UTC().Add(time.Duration(client.RefreshTokenLifetime) * time.Second)
	} else {
		t.ExpiredAt = time.Now().UTC().Add(time.Duration(client.AccessTokenLifetime) * time.Second)
	}
}
