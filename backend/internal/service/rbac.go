package service

import (
	"context"
	"errors"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"gorm.io/gorm"
)

// RBACService is the access-control contract.
type RBACService interface {
	HasPermission(ctx context.Context, userID, method, path string) (bool, error)
	UserRoutes(ctx context.Context, userID string) (map[string]bool, error)
	InvalidateUser(ctx context.Context, userID string) error
	AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	AssignRoles(ctx context.Context, userID string, roleIDs []string) error
	RoleCodes(ctx context.Context, userID string) ([]string, error)
}

// rbacService implements RBACService.
type rbacService struct {
	db    *gorm.DB
	cache *cache.Client
	users *repository.Repo[model.User]
}

// NewRBACService wires the RBAC implementation.
func NewRBACService(db *gorm.DB, c *cache.Client, users *repository.Repo[model.User]) RBACService {
	return &rbacService{db: db, cache: c, users: users}
}

// HasPermission resolves the route key "METHOD path" for a user, consulting
// the cached permission set before touching the database.
func (s *rbacService) HasPermission(ctx context.Context, userID, method, path string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	routes, err := s.UserRoutes(ctx, userID)
	if err != nil {
		return false, err
	}
	// super_admin carries a wildcard grant covering every route.
	return routes[method+" "+path] || routes["*"], nil
}

// UserRoutes returns the set of granted route keys, cached in Redis.
func (s *rbacService) UserRoutes(ctx context.Context, userID string) (map[string]bool, error) {
	cacheKey := constant.PermissionSet + userID
	if exists, err := s.cache.Exists(cacheKey); err == nil && exists {
		members, err := s.cache.SMembers(cacheKey)
		if err != nil {
			return nil, err
		}
		out := make(map[string]bool, len(members))
		for _, m := range members {
			out[m] = true
		}
		return out, nil
	}
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles").Preload("Roles.Permissions").
		First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	routes := map[string]bool{}
	for _, role := range user.Roles {
		if role.Code == constant.RoleSuperAdmin {
			routes["*"] = true
		}
		for _, p := range role.Permissions {
			routes[p.Route] = true
		}
	}
	keys := make([]string, 0, len(routes))
	for k := range routes {
		keys = append(keys, k)
	}
	if len(keys) > 0 {
		if err := s.cache.SAdd(cacheKey, keys...); err == nil {
			_ = s.cache.Expire(cacheKey, constant.PermissionTTL)
		}
	}
	return routes, nil
}

// InvalidateUser drops the cached permission set (call after role changes).
func (s *rbacService) InvalidateUser(ctx context.Context, userID string) error {
	return s.cache.Del(constant.PermissionSet + userID)
}

// AssignPermissions replaces the permission set of a role atomically and
// invalidates the cached permission sets of every user holding that role.
func (s *rbacService) AssignPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			if err := tx.Create(&model.RolePermission{RoleID: roleID, PermissionID: pid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.invalidateRoleUsers(ctx, roleID)
}

// invalidateRoleUsers drops cached permission sets for all users of a role.
func (s *rbacService) invalidateRoleUsers(ctx context.Context, roleID string) error {
	var userIDs []string
	if err := s.db.WithContext(ctx).Model(&model.UserRole{}).
		Where("role_id = ?", roleID).Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}
	for _, uid := range userIDs {
		if err := s.cache.Del(constant.PermissionSet + uid); err != nil {
			return err
		}
	}
	return nil
}

// AssignRoles replaces the roles of a user atomically.
func (s *rbacService) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		for _, rid := range roleIDs {
			if err := tx.Create(&model.UserRole{UserID: userID, RoleID: rid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.InvalidateUser(ctx, userID)
}

// RoleCodes returns the role codes assigned to a user.
func (s *rbacService) RoleCodes(ctx context.Context, userID string) ([]string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user")
		}
		return nil, err
	}
	codes := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		codes = append(codes, strings.ToLower(r.Code))
	}
	return codes, nil
}

var _ RBACService = (*rbacService)(nil)
