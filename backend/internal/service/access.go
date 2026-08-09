package service

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"gorm.io/gorm"
)

// grantPermissionModify lists the grant permissions that allow changing a
// document (as opposed to merely reading it).
var grantPermissionModify = []string{"write", "approve"}

// isSuperAdmin reports whether the user holds the super_admin role. A
// transient query error fails closed (false) so a DB blip never widens access.
func isSuperAdmin(ctx context.Context, db *gorm.DB, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	var count int64
	err := db.WithContext(ctx).Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.code = ?", userID, constant.RoleSuperAdmin).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// userRoleIDs loads the role ids assigned to the user (used to resolve
// role-based access grants).
func userRoleIDs(ctx context.Context, db *gorm.DB, userID string) []string {
	var roleIDs []string
	_ = db.WithContext(ctx).Model(&model.UserRole{}).
		Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error
	return roleIDs
}

// documentAccessScope returns a GORM scope that restricts a documents query
// to rows the caller may read: owned documents, documents with an active
// access grant (by user or by one of the user's roles) and — for super
// admins — everything. With no usable identity the scope denies all rows, so
// a mis-wired caller can never see more than intended.
//
// The caller lookups (super-admin check, role ids) run once when the scope is
// built — the returned closure is applied to several sub-queries (rows,
// count, summary), so re-running them per application would multiply the DB
// round-trips.
func documentAccessScope(ctx context.Context, db *gorm.DB, userID string) func(*gorm.DB) *gorm.DB {
	if userID == "" {
		return func(tx *gorm.DB) *gorm.DB { return tx.Where("1 = 0") }
	}
	super, err := isSuperAdmin(ctx, db, userID)
	if err != nil {
		return func(tx *gorm.DB) *gorm.DB { return tx.Where("1 = 0") } // fail closed
	}
	if super {
		return func(tx *gorm.DB) *gorm.DB { return tx }
	}
	roleIDs := userRoleIDs(ctx, db, userID)
	return func(tx *gorm.DB) *gorm.DB {
		granted := db.WithContext(ctx).Model(&model.Access{}).
			Select("document_id").
			Where("revoked_at IS NULL").
			Where("user_id = ? OR role_id IN ?", userID, roleIDs)
		return tx.Where("owner_id = ? OR id IN (?)", userID, granted)
	}
}

// documentReadableID reports whether the caller may read the document: owner,
// super admin, or an active access grant (user or role). Existence is not
// revealed — callers map a denial to the same 404 a missing row yields.
// It resolves ownership from the DB so it can be used before a cached copy
// is served.
func documentReadableID(ctx context.Context, db *gorm.DB, userID, docID string) (bool, error) {
	if userID == "" || docID == "" {
		return false, nil
	}
	super, err := isSuperAdmin(ctx, db, userID)
	if err != nil {
		return false, err
	}
	if super {
		return true, nil
	}
	granted := db.WithContext(ctx).Model(&model.Access{}).
		Select("document_id").
		Where("revoked_at IS NULL").
		Where("user_id = ? OR role_id IN ?", userID, userRoleIDs(ctx, db, userID))
	var count int64
	err = db.WithContext(ctx).Model(&model.Document{}).
		Where("id = ?", docID).
		Where("owner_id = ? OR id IN (?)", userID, granted).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// documentModifiable reports whether the caller may change the document:
// owner, super admin, or an active write/approve grant (user or role).
func documentModifiable(ctx context.Context, db *gorm.DB, userID string, doc *model.Document) (bool, error) {
	if userID == "" || doc == nil {
		return false, nil
	}
	if doc.OwnerID == userID {
		return true, nil
	}
	super, err := isSuperAdmin(ctx, db, userID)
	if err != nil {
		return false, err
	}
	if super {
		return true, nil
	}
	return activeGrantCount(ctx, db, userID, doc.ID, grantPermissionModify) > 0, nil
}

// activeGrantCount counts non-revoked access grants for the document held by
// the user directly or through their roles. When permissions is non-nil only
// grants with one of those permissions count.
func activeGrantCount(ctx context.Context, db *gorm.DB, userID, documentID string, permissions []string) int64 {
	tx := db.WithContext(ctx).Model(&model.Access{}).
		Where("document_id = ? AND revoked_at IS NULL", documentID).
		Where("user_id = ? OR role_id IN ?", userID, userRoleIDs(ctx, db, userID))
	if len(permissions) > 0 {
		tx = tx.Where("permission IN ?", permissions)
	}
	var count int64
	if err := tx.Count(&count).Error; err != nil {
		return 0 // fail closed on a transient error
	}
	return count
}
