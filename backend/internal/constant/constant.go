// Package constant holds shared keys and enums used across the backend.
package constant

import "time"

// Redis key prefixes.
const (
	// AdminToken is the single-sign-on session key: admin_token:<user_id>.
	AdminToken = "admin_token:"
	// PermissionSet caches a user's granted route keys: perm:user:<id>.
	PermissionSet = "perm:user:"
	// EntityCache is the entity cache prefix: cache:<slug>:<id>.
	EntityCache = "cache:"
	// RateLimit is the rate limiter key: ratelimit:<key>.
	RateLimit = "ratelimit:"
	// SagaState is the saga state key: saga:<saga_id>.
	SagaState = "saga:"
)

// Cache TTLs.
const (
	EntityCacheTTL   = 15 * time.Minute
	PermissionTTL    = 5 * time.Minute
	SessionTTL       = 24 * time.Hour
	RateLimitWindow  = time.Minute
)

// Built-in roles.
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
)

// Document statuses.
const (
	DocDraft            = "draft"
	DocPendingVerif     = "pending_verification"
	DocVerified         = "verified"
	DocRejected         = "rejected"
	DocApproved         = "approved"
	DocArchived         = "archived"
)

// Verification / approval statuses.
const (
	StatusPending  = "pending"
	StatusVerified = "verified"
	StatusRejected = "rejected"
	StatusApproved = "approved"
	StatusInProgress = "in_progress"
	StatusFailed   = "failed"
)

// Outbox / audit action names.
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionLogin  = "login"
	ActionLogout = "logout"
)

// WebSocket event names (README contract).
const (
	EventDocumentUploaded   = "document_uploaded"
	EventVerificationNeeded = "verification_needed"
	EventApprovalRequired   = "approval_required"
	EventDocumentReady      = "document_ready"
	EventDocumentUpdated    = "document_updated"
	EventUserRegistered     = "user_registered"
	EventAccessGranted      = "access_granted"
)
