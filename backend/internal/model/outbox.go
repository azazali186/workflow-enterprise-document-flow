package model

import "time"

// OutboxMessage is a transactional outbox row awaiting publication.
type OutboxMessage struct {
	BaseModel
	TraceID       string     `gorm:"column:trace_id;size:64;index" json:"trace_id,omitempty"`
	AggregateType string     `gorm:"column:aggregate_type;size:64;index" json:"aggregate_type"`
	AggregateID   string     `gorm:"column:aggregate_id;size:64;index" json:"aggregate_id"`
	EventType     string     `gorm:"column:event_type;size:64;index" json:"event_type"`
	Payload       string     `gorm:"column:payload;type:text" json:"payload"`
	Status        string     `gorm:"column:status;size:16;default:pending;index" json:"status"`
	Attempts      int        `gorm:"column:attempts;default:0" json:"attempts"`
	LastError     string     `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	PublishedAt   *time.Time `gorm:"column:published_at" json:"published_at,omitempty"`
}

// TableName returns the snake_case table name.
func (OutboxMessage) TableName() string { return "outbox_messages" }
