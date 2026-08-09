package model

// Permission represents an API route that may be granted to roles.
type Permission struct {
	BaseModel
	Name    string `gorm:"column:name;size:120;not null" json:"name"`
	Route   string `gorm:"column:route;size:255;uniqueIndex;not null" json:"route"` // "POST /api/v1/documents/list"
	Path    string `gorm:"column:path;size:255;index;not null" json:"path"`
	Method  string `gorm:"column:method;size:10;not null" json:"method"`
	Service string `gorm:"column:service;size:64;default:api-gateway" json:"service"`
}

// TableName returns the snake_case table name.
func (Permission) TableName() string { return "permissions" }

// RolePermission is the explicit join between roles and permissions.
type RolePermission struct {
	RoleID       string `gorm:"column:role_id;type:uuid;primaryKey" json:"role_id"`
	PermissionID string `gorm:"column:permission_id;type:uuid;primaryKey" json:"permission_id"`
}

// TableName returns the join table name.
func (RolePermission) TableName() string { return "role_permissions" }
