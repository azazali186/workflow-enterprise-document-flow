package service

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"gorm.io/gorm"
)

// AuditService records who changed what, with redacted before/after payloads.
// Sensitive fields (passwords, tokens, keys) are never written to the trail.
type AuditService struct {
	db *gorm.DB
}

// NewAuditService builds the audit recorder.
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Record writes one audit entry.
func (a *AuditService) Record(ctx context.Context, actor Actor, action, entity, entityID string, before, after any) error {
	entry := model.AuditLog{
		ActorID:    model.NullableString(actor.ID),
		ActorEmail: actor.Email,
		Action:     action,
		Entity:     entity,
		EntityID:   entityID,
		BeforeData: crypto.MarshalRedacted(before),
		AfterData:  crypto.MarshalRedacted(after),
		IPAddress:  actor.IP,
		UserAgent:  actor.UA,
	}
	return a.db.WithContext(ctx).Create(&entry).Error
}

// Change records an update with both payloads.
func (a *AuditService) Change(ctx context.Context, actor Actor, entity, entityID string, before, after any) error {
	return a.Record(ctx, actor, "update", entity, entityID, before, after)
}

// Action records a side-effect without payloads.
func (a *AuditService) Action(ctx context.Context, actor Actor, action, entity, entityID string) error {
	return a.Record(ctx, actor, action, entity, entityID, nil, nil)
}
