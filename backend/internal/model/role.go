package model

// Role groups permissions and is assigned to users via user_roles.
type Role struct {
	BaseModel
	Code        string       `gorm:"column:code;size:64;uniqueIndex;not null" json:"code"`
	Name        string       `gorm:"column:name;size:120;not null" json:"name"`
	Description string       `gorm:"column:description;size:500" json:"description,omitempty"`
	IsSystem    bool         `gorm:"column:is_system;default:false" json:"is_system"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// TableName returns the snake_case table name.
func (Role) TableName() string { return "roles" }

// UserRole is the explicit join model between users and roles.
type UserRole struct {
	UserID string `gorm:"column:user_id;type:uuid;primaryKey" json:"user_id"`
	RoleID string `gorm:"column:role_id;type:uuid;primaryKey" json:"role_id"`
}

// TableName returns the join table name.
func (UserRole) TableName() string { return "user_roles" }
