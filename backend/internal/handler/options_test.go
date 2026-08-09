package handler

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newOptionsHandler(t *testing.T, name string) *OptionsHandler {
	t.Helper()
	// Unique in-memory DB name per test (shared cache would leak rows).
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{&model.User{}, &model.Category{}} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Create(&model.User{Email: "alice@example.com", Name: "Alice Adams", Status: model.UserActive})
	_ = db.Create(&model.User{Email: "bob@example.com", Name: "Bob Brown", Status: model.UserLocked})
	_ = db.Create(&model.Category{Name: "Finance", Slug: "finance"})
	return NewOptionsHandler(service.NewOptionsService(db))
}

// TestOptionsHandlerList verifies the options endpoint returns {id, name}
// pairs filtered by search, restricted to active users.
func TestOptionsHandlerList(t *testing.T) {
	h := newOptionsHandler(t, "opthandlerlist")
	serverInstance := server.New(server.WithHostPorts("127.0.0.1:0"))
	serverInstance.POST("/api/v1/options/list", h.List)
	engine := serverInstance.Engine

	payload, _ := json.Marshal(map[string]any{"type": "users", "search": "ali", "limit": 10})
	w := ut.PerformRequest(engine, "POST", "/api/v1/options/list",
		&ut.Body{Body: bytes.NewReader(payload), Len: len(payload)},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	var resp struct {
		Code int `json:"code"`
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d: %s", resp.Code, w.Body.String())
	}
	if len(resp.Data) != 1 || resp.Data[0].Name != "Alice Adams" {
		t.Fatalf("expected just Alice Adams, got %+v", resp.Data)
	}
}

// TestOptionsHandlerUnknownType verifies unknown option types return a clean
// business error (400 in the envelope), never a 500.
func TestOptionsHandlerUnknownType(t *testing.T) {
	h := newOptionsHandler(t, "opthandlerbad")
	serverInstance := server.New(server.WithHostPorts("127.0.0.1:0"))
	serverInstance.POST("/api/v1/options/list", h.List)
	engine := serverInstance.Engine

	payload, _ := json.Marshal(map[string]any{"type": "nope"})
	w := ut.PerformRequest(engine, "POST", "/api/v1/options/list",
		&ut.Body{Body: bytes.NewReader(payload), Len: len(payload)},
		ut.Header{Key: "Content-Type", Value: "application/json"})

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 400 {
		t.Fatalf("expected business code 400, got %d: %s", resp.Code, w.Body.String())
	}
}
