package model

// LoginLog records every authentication attempt (success and failure).
type LoginLog struct {
	BaseModel
	UserID        *string `gorm:"column:user_id;type:uuid;index" json:"user_id,omitempty"`
	Email         string `gorm:"column:email;size:255;index" json:"email"`
	Status        string `gorm:"column:status;size:20;index;not null" json:"status"` // success|failure
	FailureReason string `gorm:"column:failure_reason;size:255" json:"failure_reason,omitempty"`
	IPAddress     string `gorm:"column:ip_address;size:64" json:"ip_address,omitempty"`
	UserAgent     string `gorm:"column:user_agent;size:500" json:"user_agent,omitempty"`
}

// TableName returns the snake_case table name.
func (LoginLog) TableName() string { return "login_logs" }
