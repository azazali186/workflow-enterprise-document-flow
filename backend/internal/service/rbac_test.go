package service

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func rbacHarness(t *testing.T) (RBACService, *gorm.DB, *repository.Repo[model.User]) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{
		&model.User{}, &model.Role{}, &model.UserRole{}, &model.RolePermission{},
		&model.Permission{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	users := repository.NewUserRepo(db)
	return NewRBACService(db, c, users), db, users
}

func TestSuperAdminBypassesAllRoutes(t *testing.T) {
	rbac, db, _ := rbacHarness(t)
	user := &model.User{Email: "super@example.com", PasswordHash: "x", Status: model.UserActive}
	role := &model.Role{Code: constant.RoleSuperAdmin, Name: "Super Admin", IsSystem: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(role).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	ok, err := rbac.HasPermission(context.Background(), user.ID, "POST", "/api/v1/documents/delete")
	if err != nil || !ok {
		t.Fatalf("super admin denied: ok=%v err=%v", ok, err)
	}
}

func TestRoleGrantedPermissionOnly(t *testing.T) {
	rbac, db, _ := rbacHarness(t)
	user := &model.User{Email: "u@example.com", PasswordHash: "x", Status: model.UserActive}
	role := &model.Role{Code: "editor", Name: "Editor"}
	perm := &model.Permission{Name: "List Documents", Route: "POST /api/v1/documents/list", Path: "/api/v1/documents/list", Method: "POST"}
	for _, m := range []any{user, role, perm} {
		if err := db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ok, err := rbac.HasPermission(ctx, user.ID, "POST", "/api/v1/documents/list")
	if err != nil || !ok {
		t.Fatalf("granted route denied: ok=%v err=%v", ok, err)
	}
	ok, err = rbac.HasPermission(ctx, user.ID, "POST", "/api/v1/documents/delete")
	if err != nil || ok {
		t.Fatalf("ungranted route allowed: ok=%v err=%v", ok, err)
	}
}

func TestInvalidateUserClearsCache(t *testing.T) {
	rbac, db, _ := rbacHarness(t)
	user := &model.User{Email: "c@example.com", PasswordHash: "x", Status: model.UserActive}
	role := &model.Role{Code: "viewer", Name: "Viewer"}
	perm := &model.Permission{Name: "Get", Route: "POST /api/v1/users/get", Path: "/api/v1/users/get", Method: "POST"}
	for _, m := range []any{user, role, perm} {
		if err := db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Create(&model.UserRole{UserID: user.ID, RoleID: role.ID})
	_ = db.Create(&model.RolePermission{RoleID: role.ID, PermissionID: perm.ID})
	ctx := context.Background()
	ok, err := rbac.HasPermission(ctx, user.ID, "POST", "/api/v1/users/get")
	if err != nil || !ok {
		t.Fatalf("initial grant failed: ok=%v err=%v", ok, err)
	}
	// Revoke the role then invalidate the cache.
	_ = db.Where("role_id = ?", role.ID).Delete(&model.RolePermission{})
	if err := rbac.InvalidateUser(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	ok, _ = rbac.HasPermission(ctx, user.ID, "POST", "/api/v1/users/get")
	if ok {
		t.Fatal("permission should be gone after invalidation")
	}
}
