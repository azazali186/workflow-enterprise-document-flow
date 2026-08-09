package model

import "time"

// Approval is one decision step inside an approval chain.
type Approval struct {
	BaseModel
	DocumentID  string     `gorm:"column:document_id;type:uuid;index;not null" json:"document_id"`
	Level       int        `gorm:"column:level;not null" json:"level"`
	ApproverID  string     `gorm:"column:approver_id;type:uuid;index;not null" json:"approver_id"`
	Status      string     `gorm:"column:status;size:20;default:pending;index" json:"status"`
	Comment     string     `gorm:"column:comment;type:text" json:"comment,omitempty"`
	RequestedBy string     `gorm:"column:requested_by;type:uuid;index" json:"requested_by,omitempty"`
	DecidedAt   *time.Time `gorm:"column:decided_at" json:"decided_at,omitempty"`
}

// TableName returns the snake_case table name.
func (Approval) TableName() string { return "approvals" }
