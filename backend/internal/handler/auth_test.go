package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/crypto"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/redis/go-redis/v9"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLoginHandler(t *testing.T) {
	jwt.Init("handler-test-secret", time.Hour)
	db, err := gorm.Open(sqlite.Open("file:handler?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{
		&model.User{}, &model.Role{}, &model.UserRole{}, &model.LoginLog{}, &model.AuditLog{}, &model.OutboxMessage{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Create(&model.Role{Code: constant.RoleUser, Name: "User"})

	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	users := repository.NewUserRepo(db)
	roles := repository.NewRoleRepo(db)
	logs := repository.NewLoginLogRepo(db)
	audit := service.NewAuditService(db)
	authSvc := service.NewAuthService(db, users, roles, logs, c, audit, 0)

	// Seed a user directly.
	user := &model.User{Email: "h@example.com", Name: "H", Status: model.UserActive}
	hash, _ := crypto.HashPassword("password123")
	user.PasswordHash = hash
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.POST("/api/v1/auth/login", NewAuthHandler(authSvc).Login)
	engine := h.Engine

	body, _ := json.Marshal(map[string]string{"email": "h@example.com", "password": "password123"})
	w := ut.PerformRequest(engine, "POST", "/api/v1/auth/login",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	if w.Code != 200 {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 || resp.Data.Token == "" {
		t.Fatalf("bad response: %+v", resp)
	}
}

func TestLoginHandlerRejectsBadCredentials(t *testing.T) {
	jwt.Init("handler-test-secret", time.Hour)
	db, _ := gorm.Open(sqlite.Open("file:handler2?mode=memory&cache=shared"), &gorm.Config{})
	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	users := repository.NewUserRepo(db)
	roles := repository.NewRoleRepo(db)
	logs := repository.NewLoginLogRepo(db)
	audit := service.NewAuditService(db)
	authSvc := service.NewAuthService(db, users, roles, logs, c, audit, 0)

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.POST("/api/v1/auth/login", NewAuthHandler(authSvc).Login)
	engine := h.Engine

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "x"})
	w := ut.PerformRequest(engine, "POST", "/api/v1/auth/login",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 401 {
		t.Fatalf("expected 401 code, got %d", resp.Code)
	}
}


