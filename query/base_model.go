package query

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is an embeddable base for GORM models, like gorm.Model but with
// a uint64 ID (matching every BaseRepo method that takes an ID) instead of
// gorm.Model's uint.
type BaseModel struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
