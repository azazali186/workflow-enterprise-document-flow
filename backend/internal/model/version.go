package model

// Version snapshots a document at a point in time (version control).
type Version struct {
	BaseModel
	DocumentID    string  `gorm:"column:document_id;type:uuid;index;not null" json:"document_id"`
	VersionNumber int     `gorm:"column:version_number;index;not null" json:"version_number"`
	ChangeSummary string  `gorm:"column:change_summary;type:text" json:"change_summary,omitempty"`
	Snapshot      JSONMap `gorm:"column:snapshot;type:jsonb" json:"snapshot,omitempty"`
	CreatedBy     string  `gorm:"column:created_by;type:uuid;index" json:"created_by,omitempty"`
}

// TableName returns the snake_case table name.
func (Version) TableName() string { return "versions" }
