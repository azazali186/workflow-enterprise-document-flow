//go:build integration

// Package integration exercises the full backend against real infrastructure
// (PostgreSQL, Redis, NATS JetStream). Run with:
//
//	make test-integration
//
// or:
//
//	ENV=development \
//	DATABASE_URL=postgres://aeroxe:secret@localhost:5432/docu_flow_test?sslmode=disable \
//	JWT_SECRET=integration-test-secret \
//	ENCRYPTION_KEY=$(openssl rand -base64 32) \
//	go test ./internal/integration -tags=integration -v
package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/lock"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/outbox"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
	"github.com/aeroxe/docu-flow/backend/internal/saga"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/aeroxe/docu-flow/backend/internal/worker"
	"github.com/aeroxe/docu-flow/backend/internal/ws"
	"gorm.io/gorm"
)

// resetTestTables truncates every table except the migration ledger so each
// run starts from a clean slate. CASCADE clears FK-referencing rows too.
func resetTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	var tables []string
	if err := db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'schema_migrations'").Scan(&tables).Error; err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}
	if err := db.Exec("TRUNCATE TABLE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set; skipping real-infra integration test")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func setup(t *testing.T) (*gorm.DB, *cache.Client, *saga.Coordinator) {
	t.Helper()
	cfg := testConfig(t)
	_ = logger.Init("warn", false)
	if _, err := database.Init(cfg); err != nil {
		t.Fatalf("db init: %v", err)
	}
	// Use the same versioned SQL migrations production runs, so the suite
	// validates the real schema (including future ALTERs).
	if err := database.MigrateVersioned(database.DB, cfg); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Reset app tables so repeated local runs against a persistent test DB
	// behave like CI's fresh Postgres container (the suite seeds fixed-ID
	// fixtures that would otherwise collide on re-run). schema_migrations is
	// kept so versioned migrations stay applied.
	resetTestTables(t, database.DB)
	// Long-lived context for the cache client (a canceled one would break
	// every subsequent Redis call).
	rctx := context.Background()
	if err := database.InitRedis(rctx, cfg); err != nil {
		t.Fatalf("redis init: %v", err)
	}
	if err := database.InitNATS(cfg); err != nil {
		t.Fatalf("nats init: %v", err)
	}
	c := cache.New(rctx, database.RDB)
	if err := database.SeedRBAC(database.DB, cfg); err != nil {
		t.Fatalf("seed rbac: %v", err)
	}
	return database.DB, c, saga.New(c, database.DB)
} // TestOutboxToJetStream proves the transactional outbox actually publishes a
// message to JetStream and the dispatcher marks it published.
func TestOutboxToJetStream(t *testing.T) {
	db, _, _ := setup(t)
	t.Cleanup(func() { database.CloseNATS() })

	ctx := context.Background()
	// Unique aggregate id per run so stale rows from earlier runs can't match.
	aggID := "it-" + uuidx.New()
	if err := outbox.Enqueue(ctx, db, outbox.Event{
		AggregateType: "integration", AggregateID: aggID, EventType: "integration.event",
	}, map[string]any{"hello": "world"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	lk := lock.New(database.RDB)
	disp := outbox.NewDispatcher(db, &jsPublisher{}, outbox.Options{
		Interval: 100 * time.Millisecond, Batch: 10, MaxAttempts: 3,
	})
	// Drain one cycle synchronously instead of racing the ticker.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := disp.DispatchOnce(ctx, lk); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		var m model.OutboxMessage
		if err := db.Where("aggregate_id = ?", aggID).First(&m).Error; err != nil {
			t.Fatalf("load outbox msg: %v", err)
		}
		if m.Status == outbox.StatusPublished {
			t.Logf("outbox message published: %s", m.ID)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("outbox message never reached published state")
}

// TestSagaLifecycle drives a saga through its steps against Redis and checks
// idempotency of CompleteStep.
func TestSagaLifecycle(t *testing.T) {
	db, _, co := setup(t)
	t.Cleanup(func() { database.CloseNATS() })

	ctx := context.Background()
	s, err := co.Start(ctx, "integration_saga", "document", "agg-42", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("start saga: %v", err)
	}
	if s.Status != saga.StatusRunning || s.CurrentStep != 0 {
		t.Fatalf("unexpected initial state: %+v", s)
	}
	// FindByAggregate via the index.
	found, err := co.FindByAggregate(ctx, "document", "agg-42")
	if err != nil || found.ID != s.ID {
		t.Fatalf("find by aggregate: id=%s err=%v", s.ID, err)
	}
	if _, err := co.CompleteStep(ctx, s.ID, "a"); err != nil {
		t.Fatalf("complete a: %v", err)
	}
	// Idempotency: completing "a" again must not advance the saga.
	after, err := co.CompleteStep(ctx, s.ID, "a")
	if err != nil {
		t.Fatalf("re-complete a: %v", err)
	}
	if after.CurrentStep != 1 {
		t.Fatalf("idempotent re-complete advanced saga to step %d", after.CurrentStep)
	}
	if _, err := co.CompleteStep(ctx, s.ID, "b"); err != nil {
		t.Fatalf("complete b: %v", err)
	}
	done, err := co.CompleteStep(ctx, s.ID, "c")
	if err != nil {
		t.Fatalf("complete c: %v", err)
	}
	if done.Status != saga.StatusCompleted {
		t.Fatalf("saga not completed: %+v", done)
	}
	// Outbox events for the transitions should exist.
	var count int64
	if err := db.Model(&model.OutboxMessage{}).Where("aggregate_id = ?", "agg-42").Count(&count).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count < 4 { // started + a + b + c
		t.Fatalf("expected >=4 outbox events, got %d", count)
	}
}

// TestFullPipeline wires the real services and proves the worker pipeline:
// create document → saga upload steps → finalize to pending_verification,
// all driven through the same HandleEvent path the NATS consumer uses.
func TestFullPipeline(t *testing.T) {
	db, c, co := setup(t)
	t.Cleanup(func() { database.CloseNATS() })

	ctx := context.Background()
	audit := service.NewAuditService(db)
	docRepo := repository.NewDocumentRepo(db)
	storageRepo := repository.NewStorageRepo(db)
	approvalRepo := repository.NewApprovalRepo(db)
	hub := ws.NewHub()

	docSvc := service.NewDocumentService(db, docRepo, repository.NewVersionRepo(db), audit, c, co, worker.NoopIndexer{})
	// The versioned schema enforces real FKs: the document owner must exist.
	ownerID := "10000000-0000-7000-8000-000000000001"
	if err := db.Create(&model.User{
		BaseModel:    model.BaseModel{ID: ownerID},
		Email:        "pipeline-owner@example.com",
		PasswordHash: "x",
		Status:       model.UserActive,
	}).Error; err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	doc, err := docSvc.Create(ctx, service.Actor{ID: ownerID}, service.CreateDocumentInput{
		Title: "Integration Doc", Description: "proves the pipeline", OwnerID: ownerID,
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	if doc.Status != "draft" {
		t.Fatalf("expected draft, got %s", doc.Status)
	}
	// The saga must exist and be running.
	s, err := co.FindByAggregate(ctx, "document", doc.ID)
	if err != nil || s.Name != saga.DocumentUpload {
		t.Fatalf("saga missing after create: err=%v", err)
	}

	w := worker.New(db, database.JS, c, co, hub, docRepo, storageRepo, approvalRepo, audit, worker.Options{})
	// Drive the pipeline through the exact events the outbox publishes.
	events := []string{
		"document_uploaded",
		"saga.upload", "saga.virus_scan", "saga.metadata_extraction", "saga.storage",
	}
	for _, evtType := range events {
		if err := w.HandleEvent(ctx, outbox.Event{
			AggregateType: "document", AggregateID: doc.ID, EventType: evtType,
		}); err != nil {
			t.Fatalf("handle %s: %v", evtType, err)
		}
	}
	var reloaded model.Document
	if err := db.First(&reloaded, "id = ?", doc.ID).Error; err != nil {
		t.Fatalf("reload document: %v", err)
	}
	if reloaded.Status != "pending_verification" {
		t.Fatalf("worker did not finalize document, status=%s", reloaded.Status)
	}
	meta := reloaded.Meta
	if meta["indexed_at"] == nil || meta["title_length"] == nil {
		t.Fatalf("worker did not enrich metadata: %+v", meta)
	}
	final, err := co.FindByAggregate(ctx, "document", doc.ID)
	if err != nil || final.Status != saga.StatusCompleted {
		t.Fatalf("saga not completed: status=%+v err=%v", final, err)
	}
}

// TestVerificationAndApprovalSagas proves the README VerificationProcess and
// ApprovalChain sagas are orchestrated end to end: they start when the
// workflow is created and complete when the decision lands.
func TestVerificationAndApprovalSagas(t *testing.T) {
	db, c, co := setup(t)
	t.Cleanup(func() { database.CloseNATS() })

	ctx := context.Background()
	audit := service.NewAuditService(db)
	docRepo := repository.NewDocumentRepo(db)
	verifRepo := repository.NewVerificationRepo(db)
	approvalRepo := repository.NewApprovalRepo(db)
	lk := lock.New(database.RDB)
	hub := ws.NewHub()

	docSvc := service.NewDocumentService(db, docRepo, repository.NewVersionRepo(db), audit, c, co, worker.NoopIndexer{})
	ownerID := "20000000-0000-7000-8000-000000000001"
	approverID := "20000000-0000-7000-8000-000000000002"
	approverID2 := "20000000-0000-7000-8000-000000000003"
	for _, u := range []*model.User{
		{BaseModel: model.BaseModel{ID: ownerID}, Email: "saga-owner@example.com", PasswordHash: "x", Status: model.UserActive},
		{BaseModel: model.BaseModel{ID: approverID}, Email: "saga-approver@example.com", PasswordHash: "x", Status: model.UserActive},
		{BaseModel: model.BaseModel{ID: approverID2}, Email: "saga-approver2@example.com", PasswordHash: "x", Status: model.UserActive},
	} {
		if err := db.Create(u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	doc, err := docSvc.Create(ctx, service.Actor{ID: ownerID}, service.CreateDocumentInput{
		Title: "Saga Doc", Description: "verification + approval sagas", OwnerID: ownerID,
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	// Drive the upload pipeline to completion first.
	w := worker.New(db, database.JS, c, co, hub, docRepo, repository.NewStorageRepo(db), approvalRepo, audit, worker.Options{})
	for _, evtType := range []string{"document_uploaded", "saga.upload", "saga.virus_scan",
		"saga.metadata_extraction", "saga.storage"} {
		if err := w.HandleEvent(ctx, outbox.Event{
			AggregateType: "document", AggregateID: doc.ID, EventType: evtType,
		}); err != nil {
			t.Fatalf("handle %s: %v", evtType, err)
		}
	}

	// VerificationProcess: starts on create, completes on decision.
	verifSvc := service.NewVerificationService(db, verifRepo, docRepo, audit, lk, co)
	ver, err := verifSvc.Create(ctx, service.Actor{ID: ownerID}, service.CreateVerificationInput{
		DocumentID: doc.ID, Method: "manual",
	})
	if err != nil {
		t.Fatalf("create verification: %v", err)
	}
	vs, err := co.FindByAggregate(ctx, "verification", ver.ID)
	if err != nil || vs.Name != saga.VerificationProcess || vs.Status != saga.StatusRunning {
		t.Fatalf("verification saga missing/running: %+v err=%v", vs, err)
	}
	if _, err := verifSvc.Decide(ctx, service.Actor{ID: ownerID}, service.DecideVerificationInput{
		VerificationID: ver.ID, Decision: "verified",
	}); err != nil {
		t.Fatalf("decide verification: %v", err)
	}
	vs, err = co.FindByAggregate(ctx, "verification", ver.ID)
	if err != nil || vs.Status != saga.StatusCompleted {
		t.Fatalf("verification saga not completed: %+v err=%v", vs, err)
	}

	// ApprovalChain: routing completes at create, decision completes only when
	// every level is decided (two approvers prove the saga stays running after
	// the first decision).
	approvalSvc := service.NewApprovalService(db, approvalRepo, docRepo, audit, lk, co)
	if _, err := approvalSvc.CreateChain(ctx, service.Actor{ID: ownerID}, service.CreateApprovalInput{
		DocumentID: doc.ID, ApproverIDs: []string{approverID, approverID2},
	}); err != nil {
		t.Fatalf("create approval chain: %v", err)
	}
	as, err := co.FindByAggregate(ctx, "approval", doc.ID)
	if err != nil || as.Name != saga.ApprovalChain {
		t.Fatalf("approval saga missing: %+v err=%v", as, err)
	}
	if as.CurrentStep != 1 { // routing done, decision running
		t.Fatalf("approval saga should be on decision step, got %d", as.CurrentStep)
	}
	var apprs []model.Approval
	if err := db.Where("document_id = ?", doc.ID).Order("level").Find(&apprs).Error; err != nil {
		t.Fatalf("load approvals: %v", err)
	}
	if len(apprs) != 2 {
		t.Fatalf("expected 2 approvals, got %d", len(apprs))
	}
	// First decision: the chain is still open, so the saga must keep running.
	if _, err := approvalSvc.Decide(ctx, service.Actor{ID: approverID}, service.DecideApprovalInput{
		ApprovalID: apprs[0].ID, Decision: "approved",
	}); err != nil {
		t.Fatalf("decide approval 1: %v", err)
	}
	as, err = co.FindByAggregate(ctx, "approval", doc.ID)
	if err != nil || as.Status != saga.StatusRunning {
		t.Fatalf("approval saga completed prematurely: %+v err=%v", as, err)
	}
	// Second decision resolves the chain and completes the saga.
	if _, err := approvalSvc.Decide(ctx, service.Actor{ID: approverID2}, service.DecideApprovalInput{
		ApprovalID: apprs[1].ID, Decision: "approved",
	}); err != nil {
		t.Fatalf("decide approval 2: %v", err)
	}
	as, err = co.FindByAggregate(ctx, "approval", doc.ID)
	if err != nil || as.Status != saga.StatusCompleted {
		t.Fatalf("approval saga not completed: %+v err=%v", as, err)
	}
	var reloaded model.Document
	if err := db.First(&reloaded, "id = ?", doc.ID).Error; err != nil {
		t.Fatalf("reload document: %v", err)
	}
	if reloaded.Status != "approved" {
		t.Fatalf("document status = %s, want approved", reloaded.Status)
	}
}

// jsPublisher adapts JetStream to the outbox Publisher interface.
type jsPublisher struct{}

func (j *jsPublisher) Publish(subject string, data []byte) error {
	_, err := database.JS.Publish(subject, data)
	return err
}
