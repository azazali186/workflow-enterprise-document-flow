package model

import "time"

// Storage tracks a stored document binary. Sensitive fields (object_key)
// are encrypted at rest before persisting.
type Storage struct {
	BaseModel
	DocumentID string     `gorm:"column:document_id;type:uuid;index;not null" json:"document_id"`
	Provider   string     `gorm:"column:provider;size:32;default:local" json:"provider"`
	Bucket     string     `gorm:"column:bucket;size:120" json:"bucket,omitempty"`
	ObjectKey  string     `gorm:"column:object_key;size:500" json:"-"` // encrypted
	FileName   string     `gorm:"column:file_name;size:255" json:"file_name,omitempty"`
	MimeType   string     `gorm:"column:mime_type;size:120" json:"mime_type,omitempty"`
	SizeBytes  int64      `gorm:"column:size_bytes;default:0" json:"size_bytes"`
	Checksum   string     `gorm:"column:checksum;size:64" json:"checksum,omitempty"`
	Status     string     `gorm:"column:status;size:20;default:stored;index" json:"status"`
	StoredAt   *time.Time `gorm:"column:stored_at" json:"stored_at,omitempty"`
}

// TableName returns the snake_case table name.
func (Storage) TableName() string { return "storages" }
