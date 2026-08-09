package model

import "time"

// UserStatus values.
const (
	UserActive  = "active"
	UserLocked  = "locked"
	UserPending = "pending"
)

// User represents an account in the system.
type User struct {
	BaseModel
	Email        string     `gorm:"column:email;size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"column:password_hash;size:255;not null" json:"-"`
	Name         string     `gorm:"column:name;size:120" json:"name"`
	Phone        string     `gorm:"column:phone;size:255" json:"phone,omitempty"`
	Status       string     `gorm:"column:status;size:20;default:active;index" json:"status"`
	AvatarURL    string     `gorm:"column:avatar_url;size:500" json:"avatar_url,omitempty"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	Roles        []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// TableName sets the explicit snake_case table name.
func (User) TableName() string { return "users" }
