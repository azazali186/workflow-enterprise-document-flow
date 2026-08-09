// Package model defines the GORM data model. Every table uses a UUID v7
// primary key, snake_case columns, timestamps and soft delete.
package model

import (
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"gorm.io/gorm"
)

// BaseModel is embedded by every entity.
type BaseModel struct {
	ID        string         `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"column:created_at;index" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`
}

// BeforeCreate assigns a UUID v7 when the caller did not provide one.
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuidx.New()
	}
	return nil
}

// TableName overrides for models whose pluralisation is irregular.
func (BaseModel) TableName() string { return "" }
