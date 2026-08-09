package repository

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"gorm.io/gorm"
)

// NewUserRepo builds the users repository.
func NewUserRepo(db *gorm.DB) *Repo[model.User] {
	r := New[model.User](db,
		map[string]string{"created_at": "created_at", "email": "email", "name": "name", "status": "status"},
		map[string]string{"status": "status", "email": "email", "name": "name"},
		[]string{"email", "name"},
	)
	r.Preloads = []string{"Roles"}
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByStatus(ctx, r.scope(q))
	}
	return r
}

// NewDocumentRepo builds the documents repository.
func NewDocumentRepo(db *gorm.DB) *Repo[model.Document] {
	r := New[model.Document](db,
		map[string]string{"created_at": "created_at", "updated_at": "updated_at", "title": "title",
			"status": "status", "document_number": "document_number", "size_bytes": "size_bytes"},
		map[string]string{"status": "status", "category_id": "category_id", "owner_id": "owner_id"},
		[]string{"title", "document_number", "description"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		summary, err := countByStatus(ctx, r.scope(q))
		if err != nil {
			return nil, err
		}
		var agg struct {
			TotalBytes int64 `gorm:"column:total_bytes"`
		}
		if err := r.scope(q).Select("COALESCE(SUM(size_bytes),0) AS total_bytes").Scan(&agg).Error; err != nil {
			return nil, err
		}
		summary["total_size_bytes"] = agg.TotalBytes
		return summary, nil
	}
	return r
}

// NewCategoryRepo builds the categories repository.
func NewCategoryRepo(db *gorm.DB) *Repo[model.Category] {
	r := New[model.Category](db,
		map[string]string{"created_at": "created_at", "name": "name", "sort_order": "sort_order"},
		map[string]string{"is_active": "is_active", "parent_id": "parent_id"},
		[]string{"name", "slug"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByBool(ctx, r.scope(q), "is_active")
	}
	return r
}

// NewVersionRepo builds the versions repository (scoped per document).
func NewVersionRepo(db *gorm.DB) *Repo[model.Version] {
	r := New[model.Version](db,
		map[string]string{"created_at": "created_at", "version_number": "version_number"},
		map[string]string{"document_id": "document_id"},
		nil,
	)
	return r
}

// NewTemplateRepo builds the templates repository.
func NewTemplateRepo(db *gorm.DB) *Repo[model.Template] {
	r := New[model.Template](db,
		map[string]string{"created_at": "created_at", "name": "name", "version": "version"},
		map[string]string{"is_active": "is_active", "category_id": "category_id"},
		[]string{"name", "slug"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByBool(ctx, r.scope(q), "is_active")
	}
	return r
}

// NewStorageRepo builds the storages repository.
func NewStorageRepo(db *gorm.DB) *Repo[model.Storage] {
	r := New[model.Storage](db,
		map[string]string{"created_at": "created_at", "size_bytes": "size_bytes", "status": "status"},
		map[string]string{"status": "status", "document_id": "document_id", "provider": "provider"},
		nil,
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		summary, err := countByStatus(ctx, r.scope(q))
		if err != nil {
			return nil, err
		}
		var agg struct {
			TotalBytes int64 `gorm:"column:total_bytes"`
		}
		if err := r.scope(q).Select("COALESCE(SUM(size_bytes),0) AS total_bytes").Scan(&agg).Error; err != nil {
			return nil, err
		}
		summary["total_size_bytes"] = agg.TotalBytes
		return summary, nil
	}
	return r
}

// NewVerificationRepo builds the verifications repository.
func NewVerificationRepo(db *gorm.DB) *Repo[model.Verification] {
	r := New[model.Verification](db,
		map[string]string{"created_at": "created_at", "status": "status", "method": "method"},
		map[string]string{"status": "status", "document_id": "document_id", "verified_by": "verified_by"},
		nil,
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByStatus(ctx, r.scope(q))
	}
	return r
}

// NewApprovalRepo builds the approvals repository.
func NewApprovalRepo(db *gorm.DB) *Repo[model.Approval] {
	r := New[model.Approval](db,
		map[string]string{"created_at": "created_at", "level": "level", "status": "status"},
		map[string]string{"status": "status", "document_id": "document_id", "approver_id": "approver_id"},
		nil,
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByStatus(ctx, r.scope(q))
	}
	return r
}

// NewAccessRepo builds the accesses repository.
func NewAccessRepo(db *gorm.DB) *Repo[model.Access] {
	r := New[model.Access](db,
		map[string]string{"created_at": "created_at", "permission": "permission"},
		map[string]string{"document_id": "document_id", "user_id": "user_id", "permission": "permission"},
		nil,
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		var total, active int64
		sc := r.scope(q)
		if err := sc.Count(&total).Error; err != nil {
			return nil, err
		}
		if err := sc.Where("revoked_at IS NULL").Count(&active).Error; err != nil {
			return nil, err
		}
		return map[string]any{"total": total, "active": active}, nil
	}
	return r
}

// NewLoginLogRepo builds the login logs repository.
func NewLoginLogRepo(db *gorm.DB) *Repo[model.LoginLog] {
	r := New[model.LoginLog](db,
		map[string]string{"created_at": "created_at", "status": "status"},
		map[string]string{"status": "status", "email": "email", "user_id": "user_id"},
		[]string{"email"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByStatus(ctx, r.scope(q))
	}
	return r
}

// NewAuditLogRepo builds the audit logs repository.
func NewAuditLogRepo(db *gorm.DB) *Repo[model.AuditLog] {
	r := New[model.AuditLog](db,
		map[string]string{"created_at": "created_at", "action": "action", "entity": "entity"},
		map[string]string{"action": "action", "entity": "entity", "entity_id": "entity_id", "actor_id": "actor_id"},
		[]string{"actor_email", "entity"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		return countByAction(ctx, r.scope(q))
	}
	return r
}

// NewRoleRepo builds the roles repository.
func NewRoleRepo(db *gorm.DB) *Repo[model.Role] {
	r := New[model.Role](db,
		map[string]string{"created_at": "created_at", "name": "name", "code": "code"},
		map[string]string{"code": "code", "is_system": "is_system"},
		[]string{"name", "code"},
	)
	r.Preloads = []string{"Permissions"}
	return r
}

// NewPermissionRepo builds the permissions repository.
func NewPermissionRepo(db *gorm.DB) *Repo[model.Permission] {
	r := New[model.Permission](db,
		map[string]string{"created_at": "created_at", "name": "name", "path": "path", "method": "method"},
		map[string]string{"method": "method", "service": "service"},
		[]string{"name", "path", "route"},
	)
	r.SummaryFn = func(ctx context.Context, q ListQuery) (map[string]any, error) {
		var total int64
		sc := r.scope(q)
		if err := sc.Count(&total).Error; err != nil {
			return nil, err
		}
		var byMethod []struct {
			Method string `gorm:"column:method"`
			Cnt    int64  `gorm:"column:cnt"`
		}
		if err := sc.Select("method, COUNT(*) AS cnt").Group("method").Scan(&byMethod).Error; err != nil {
			return nil, err
		}
		m := map[string]any{"total": total}
		for _, row := range byMethod {
			m[row.Method] = row.Cnt
		}
		return m, nil
	}
	return r
}

func (r *Repo[T]) scope(q ListQuery) *gorm.DB {
	tx := r.DB.Model(new(T))
	if r.BaseScope != nil {
		tx = r.BaseScope(tx)
	}
	for _, sc := range q.Scopes {
		tx = sc(tx)
	}
	return tx
}
