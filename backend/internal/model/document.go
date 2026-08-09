package model

import "time"

// Document is the core entity of DocuFlow.
type Document struct {
	BaseModel
	DocumentNumber string          `gorm:"column:document_number;size:64;uniqueIndex;not null" json:"document_number"`
	Title          string          `gorm:"column:title;size:255;not null" json:"title"`
	Description    string          `gorm:"column:description;type:text" json:"description,omitempty"`
	CategoryID     *string         `gorm:"column:category_id;type:uuid;index" json:"category_id,omitempty"`
	OwnerID        string          `gorm:"column:owner_id;type:uuid;index;not null" json:"owner_id"`
	Status         string          `gorm:"column:status;size:32;default:draft;index" json:"status"`
	FileID         *string         `gorm:"column:file_id;type:uuid" json:"file_id,omitempty"`
	FileName       string          `gorm:"column:file_name;size:255" json:"file_name,omitempty"`
	MimeType       string          `gorm:"column:mime_type;size:120" json:"mime_type,omitempty"`
	SizeBytes      int64           `gorm:"column:size_bytes;default:0" json:"size_bytes"`
	ContentHash    string          `gorm:"column:content_hash;size:64" json:"content_hash,omitempty"`
	Meta           JSONMap         `gorm:"column:meta;type:jsonb" json:"meta,omitempty"`
	Tags           JSONArray       `gorm:"column:tags;type:jsonb" json:"tags,omitempty"`
	VerifiedAt     *time.Time      `gorm:"column:verified_at" json:"verified_at,omitempty"`
	ApprovedAt     *time.Time      `gorm:"column:approved_at" json:"approved_at,omitempty"`
	ArchivedAt     *time.Time      `gorm:"column:archived_at" json:"archived_at,omitempty"`
}

// TableName returns the snake_case table name.
func (Document) TableName() string { return "documents" }
