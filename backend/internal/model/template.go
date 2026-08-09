package model

// Template is a reusable document structure.
type Template struct {
	BaseModel
	Name        string `gorm:"column:name;size:120;not null;index" json:"name"`
	Slug        string `gorm:"column:slug;size:120;uniqueIndex;not null" json:"slug"`
	Description string `gorm:"column:description;size:500" json:"description,omitempty"`
	CategoryID  *string `gorm:"column:category_id;type:uuid;index" json:"category_id,omitempty"`
	Content     string `gorm:"column:content;type:text" json:"content,omitempty"`
	Version     int    `gorm:"column:version;default:1" json:"version"`
	IsActive    bool   `gorm:"column:is_active;index" json:"is_active"`
	CreatedBy   string `gorm:"column:created_by;type:uuid;index" json:"created_by,omitempty"`
}

// TableName returns the snake_case table name.
func (Template) TableName() string { return "templates" }
