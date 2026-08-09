package model

import "time"

// Verification tracks document authenticity checks.
type Verification struct {
	BaseModel
	DocumentID  string `gorm:"column:document_id;type:uuid;index;not null" json:"document_id"`
	RequestedBy string `gorm:"column:requested_by;type:uuid;index" json:"requested_by,omitempty"`
	// VerifiedBy is a pointer so an undecided verification stores NULL, not
	// an empty string (Postgres uuid columns reject '').
	VerifiedBy *string    `gorm:"column:verified_by;type:uuid;index" json:"verified_by,omitempty"`
	Status     string     `gorm:"column:status;size:20;default:pending;index" json:"status"`
	Method     string     `gorm:"column:method;size:32;default:manual" json:"method"`
	Notes      string     `gorm:"column:notes;type:text" json:"notes,omitempty"`
	Result     JSONMap    `gorm:"column:result;type:jsonb" json:"result,omitempty"`
	VerifiedAt *time.Time `gorm:"column:verified_at" json:"verified_at,omitempty"`
}

// TableName returns the snake_case table name.
func (Verification) TableName() string { return "verifications" }
