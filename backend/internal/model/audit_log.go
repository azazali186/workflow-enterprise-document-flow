package model

// AuditLog records who changed what, with redacted before/after payloads.
type AuditLog struct {
	BaseModel
	ActorID    *string `gorm:"column:actor_id;type:uuid;index" json:"actor_id,omitempty"`
	ActorEmail string `gorm:"column:actor_email;size:255;index" json:"actor_email,omitempty"`
	Action     string `gorm:"column:action;size:32;index;not null" json:"action"`
	Entity     string `gorm:"column:entity;size:64;index;not null" json:"entity"`
	EntityID   string `gorm:"column:entity_id;size:64;index" json:"entity_id,omitempty"`
	BeforeData string `gorm:"column:before_data;type:text" json:"before_data,omitempty"` // redacted JSON
	AfterData  string `gorm:"column:after_data;type:text" json:"after_data,omitempty"`   // redacted JSON
	IPAddress  string `gorm:"column:ip_address;size:64" json:"ip_address,omitempty"`
	UserAgent  string `gorm:"column:user_agent;size:500" json:"user_agent,omitempty"`
}

// TableName returns the snake_case table name.
func (AuditLog) TableName() string { return "audit_logs" }
