package model

// Category classifies documents.
type Category struct {
	BaseModel
	Name        string  `gorm:"column:name;size:120;not null;index" json:"name"`
	Slug        string  `gorm:"column:slug;size:120;uniqueIndex;not null" json:"slug"`
	Description string  `gorm:"column:description;size:500" json:"description,omitempty"`
	ParentID    *string `gorm:"column:parent_id;type:uuid;index" json:"parent_id,omitempty"`
	SortOrder   int     `gorm:"column:sort_order;default:0" json:"sort_order"`
	IsActive    bool    `gorm:"column:is_active;index" json:"is_active"`
	Children    []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// TableName returns the snake_case table name.
func (Category) TableName() string { return "categories" }
