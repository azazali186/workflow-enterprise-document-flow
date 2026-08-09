package service

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type authHarness struct {
	svc  AuthService
	db   *gorm.DB
	c    *cache.Client
	logs *repository.Repo[model.LoginLog]
}

func newAuthHarness(t *testing.T) *authHarness {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	models := []any{
		&model.User{}, &model.Role{}, &model.UserRole{}, &model.RolePermission{},
		&model.Permission{}, &model.LoginLog{}, &model.AuditLog{}, &model.OutboxMessage{},
	}
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	// Seed the built-in user role.
	if err := db.Create(&model.Role{Code: constant.RoleUser, Name: "User"}).Error; err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	users := repository.NewUserRepo(db)
	roles := repository.NewRoleRepo(db)
	logs := repository.NewLoginLogRepo(db)
	audit := NewAuditService(db)
	return &authHarness{
		svc: NewAuthService(db, users, roles, logs, c, audit, 0),
		db:  db, c: c, logs: logs,
	}
}

func TestRegisterCreatesUserAndSession(t *testing.T) {
	h := newAuthHarness(t)
	jwt.Init("unit-test-secret", time.Hour)

	res, err := h.svc.Register(context.Background(), Actor{}, RegisterInput{
		Email: "Alice@Example.com", Password: "password123", Name: "Alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" || res.User.Email != "alice@example.com" {
		t.Fatalf("bad register result: %+v", res)
	}
	// Session must be verifiable via the SSO fingerprint.
	claims, err := jwt.ParseToken(res.Token)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := RenewIfNeeded(h.c, claims.UserID, SSOValue(res.Token, claims.UserID), 0)
	if err != nil || !ok {
		t.Fatalf("sso session not established: ok=%v err=%v", ok, err)
	}
	// A login log row should exist for the auto-login.
	var logs int64
	h.db.Model(&model.LoginLog{}).Count(&logs)
	if logs < 1 {
		t.Error("expected at least one login log")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	h := newAuthHarness(t)
	jwt.Init("unit-test-secret", time.Hour)
	_, err := h.svc.Register(context.Background(), Actor{}, RegisterInput{
		Email: "dup@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.svc.Register(context.Background(), Actor{}, RegisterInput{
		Email: "dup@example.com", Password: "password123",
	})
	if ae, ok := err.(*apperror.Error); !ok || ae.Code != apperror.CodeConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestLoginSuccessAndFailure(t *testing.T) {
	h := newAuthHarness(t)
	jwt.Init("unit-test-secret", time.Hour)
	_, err := h.svc.Register(context.Background(), Actor{}, RegisterInput{
		Email: "log@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Wrong password → unauthorized + failure log.
	_, err = h.svc.Login(context.Background(), LoginInput{Email: "log@example.com", Password: "nope"}, "1.2.3.4", "ua")
	if ae, ok := err.(*apperror.Error); !ok || ae.Code != apperror.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
	var failed int64
	h.db.Model(&model.LoginLog{}).Where("status = ?", "failure").Count(&failed)
	if failed != 1 {
		t.Errorf("failure logs = %d, want 1", failed)
	}
	// Correct password → token + success log.
	res, err := h.svc.Login(context.Background(), LoginInput{Email: "log@example.com", Password: "password123"}, "1.2.3.4", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" {
		t.Fatal("no token returned")
	}
	var succeeded int64
	h.db.Model(&model.LoginLog{}).Where("status = ?", "success").Count(&succeeded)
	if succeeded < 1 {
		t.Errorf("success logs = %d", succeeded)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	h := newAuthHarness(t)
	jwt.Init("unit-test-secret", time.Hour)
	res, err := h.svc.Register(context.Background(), Actor{}, RegisterInput{
		Email: "out@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, _ := jwt.ParseToken(res.Token)
	if err := h.svc.Logout(context.Background(), Actor{}, claims.UserID); err != nil {
		t.Fatal(err)
	}
	ok, err := RenewIfNeeded(h.c, claims.UserID, SSOValue(res.Token, claims.UserID), 0)
	if err == nil || ok {
		t.Fatalf("session should be gone after logout: ok=%v err=%v", ok, err)
	}
}
