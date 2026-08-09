package model

import "time"

// Access grants a user or role permission on a document.
type Access struct {
	BaseModel
	DocumentID  string     `gorm:"column:document_id;type:uuid;index;not null" json:"document_id"`
	UserID      *string    `gorm:"column:user_id;type:uuid;index" json:"user_id,omitempty"`
	RoleID      *string    `gorm:"column:role_id;type:uuid;index" json:"role_id,omitempty"`
	Permission  string     `gorm:"column:permission;size:20;default:read" json:"permission"` // read|write|approve
	GrantedBy   string     `gorm:"column:granted_by;type:uuid;index" json:"granted_by,omitempty"`
	RevokedAt   *time.Time `gorm:"column:revoked_at;index" json:"revoked_at,omitempty"`
}

// TableName returns the snake_case table name.
func (Access) TableName() string { return "accesses" }
