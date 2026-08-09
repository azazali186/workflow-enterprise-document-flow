package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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
	"github.com/cloudwego/hertz/pkg/protocol"
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
	h.POST("/api/v1/auth/login", NewAuthHandler(authSvc, false).Login)
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

// TestRefreshHandler verifies the refresh endpoint parses the bearer token
// directly from the header. The route is public (no auth middleware runs), so
// the handler must not rely on middleware-set context values.
func TestRefreshHandler(t *testing.T) {
	jwt.Init("handler-test-secret", time.Hour)
	db, err := gorm.Open(sqlite.Open("file:handler3?mode=memory&cache=shared"), &gorm.Config{
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

	mr := miniredis.RunT(t)
	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	users := repository.NewUserRepo(db)
	roles := repository.NewRoleRepo(db)
	logs := repository.NewLoginLogRepo(db)
	audit := service.NewAuditService(db)
	authSvc := service.NewAuthService(db, users, roles, logs, c, audit, time.Hour)

	// Seed a user and establish a session like Login would.
	user := &model.User{Email: "refresh@example.com", Name: "R", Status: model.UserActive}
	hash, _ := crypto.HashPassword("password123")
	user.PasswordHash = hash
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	token, _, err := jwt.Generate(user.ID, user.Email, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set(service.SSOKey(user.ID), service.SSOValue(token, user.ID), time.Hour); err != nil {
		t.Fatal(err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.POST("/api/v1/auth/refresh", NewAuthHandler(authSvc, false).Refresh)
	engine := h.Engine

	// With a valid bearer token: expect a fresh token.
	w := ut.PerformRequest(engine, "POST", "/api/v1/auth/refresh", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token})
	var okResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &okResp); err != nil {
		t.Fatal(err)
	}
	if okResp.Code != 0 || okResp.Data.Token == "" {
		t.Fatalf("refresh with token failed: %+v", okResp)
	}

	// Without a bearer token: must 401 (the old code path read middleware-only
	// context and returned an empty user id, producing a UUID error instead).
	w2 := ut.PerformRequest(engine, "POST", "/api/v1/auth/refresh", nil)
	var badResp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &badResp); err != nil {
		t.Fatal(err)
	}
	if badResp.Code != 401 {
		t.Fatalf("refresh without token: expected 401, got %d", badResp.Code)
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
	h.POST("/api/v1/auth/login", NewAuthHandler(authSvc, false).Login)
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

// TestSessionCookies verifies login sets the HttpOnly token cookie plus the
// JS-readable CSRF cookie, refresh works via the cookie alone (no bearer
// header), and logout expires both.
func TestSessionCookies(t *testing.T) {
	jwt.Init("handler-test-secret", time.Hour)
	db, err := gorm.Open(sqlite.Open("file:handlercookies?mode=memory&cache=shared"), &gorm.Config{
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
	authSvc := service.NewAuthService(db, repository.NewUserRepo(db), repository.NewRoleRepo(db),
		repository.NewLoginLogRepo(db), c, service.NewAuditService(db), time.Hour)

	user := &model.User{Email: "cookies@example.com", Name: "C", Status: model.UserActive}
	hash, _ := crypto.HashPassword("password123")
	user.PasswordHash = hash
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}

	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	auth := NewAuthHandler(authSvc, false)
	h.POST("/api/v1/auth/login", auth.Login)
	h.POST("/api/v1/auth/refresh", auth.Refresh)
	h.POST("/api/v1/auth/logout", auth.Logout)
	engine := h.Engine

	body, _ := json.Marshal(map[string]string{"email": "cookies@example.com", "password": "password123"})
	w := ut.PerformRequest(engine, "POST", "/api/v1/auth/login",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
			CSRF  string `json:"csrf"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatal(err)
	}
	if loginResp.Code != 0 {
		t.Fatalf("login failed: %+v", loginResp)
	}
	if loginResp.Data.CSRF == "" {
		t.Fatal("login response must carry the CSRF value")
	}

	// Exactly one cookie: HttpOnly, SameSite=Lax (hertz cannot emit multiple
	// Set-Cookie headers safely, so the CSRF value travels in the body).
	cookies := peekAllStrings(w.Header(), "Set-Cookie")
	if len(cookies) != 1 {
		t.Fatalf("expected 1 Set-Cookie header, got %d: %v", len(cookies), cookies)
	}
	tokenCookie := cookies[0]
	if !strings.Contains(tokenCookie, "docuflow_token=") || !strings.Contains(tokenCookie, "HttpOnly") {
		t.Fatalf("token cookie missing HttpOnly: %q", tokenCookie)
	}
	if !strings.Contains(tokenCookie, "SameSite=Lax") {
		t.Fatalf("token cookie missing SameSite=Lax: %q", tokenCookie)
	}

	// Refresh using only the session cookie (no Authorization header).
	w2 := ut.PerformRequest(engine, "POST", "/api/v1/auth/refresh", nil,
		ut.Header{Key: "Cookie", Value: "docuflow_token=" + loginResp.Data.Token})
	var refreshResp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &refreshResp); err != nil {
		t.Fatal(err)
	}
	if refreshResp.Code != 0 {
		t.Fatalf("cookie refresh failed: %+v", refreshResp)
	}

	// Logout expires the session cookie.
	w3 := ut.PerformRequest(engine, "POST", "/api/v1/auth/logout", nil,
		ut.Header{Key: "Cookie", Value: "docuflow_token=" + loginResp.Data.Token})
	logoutCookies := peekAllStrings(w3.Header(), "Set-Cookie")
	if len(logoutCookies) != 1 {
		t.Fatalf("logout should clear 1 cookie, got %v", logoutCookies)
	}
	for _, ck := range logoutCookies {
		if !strings.Contains(ck, "Max-Age=0") && !strings.Contains(ck, "expires=") {
			t.Fatalf("logout cookie not expired: %q", ck)
		}
	}
}

// peekAllStrings converts a hertz multi-value header into strings.
func peekAllStrings(h *protocol.ResponseHeader, key string) []string {
	raw := h.PeekAll(key)
	out := make([]string, 0, len(raw))
	for _, b := range raw {
		out = append(out, string(b))
	}
	return out
}


