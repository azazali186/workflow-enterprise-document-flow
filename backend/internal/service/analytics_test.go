package service

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func analyticsHarness(t *testing.T) (AnalyticsService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []any{
		&model.Document{}, &model.Category{}, &model.Storage{},
		&model.Verification{}, &model.Approval{},
	} {
		if err := db.AutoMigrate(m); err != nil {
			t.Fatal(err)
		}
	}
	return NewAnalyticsService(db), db
}

func seedAnalyticsData(t *testing.T, db *gorm.DB) {
	t.Helper()
	owner := "00000000-0000-0000-0000-000000000001"
	legal := &model.Category{Name: "Legal", Slug: "legal", IsActive: true}
	hr := &model.Category{Name: "HR", Slug: "hr", IsActive: true}
	if err := db.Create(legal).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(hr).Error; err != nil {
		t.Fatal(err)
	}
	docs := []*model.Document{
		{DocumentNumber: "DOC-A", Title: "Contract A", OwnerID: owner, Status: constant.DocDraft, CategoryID: &legal.ID},
		{DocumentNumber: "DOC-B", Title: "Contract B", OwnerID: owner, Status: constant.DocVerified, CategoryID: &legal.ID},
		{DocumentNumber: "DOC-C", Title: "Memo C", OwnerID: owner, Status: constant.DocApproved, CategoryID: &hr.ID},
		{DocumentNumber: "DOC-D", Title: "Note D", OwnerID: owner, Status: constant.DocDraft}, // uncategorized
	}
	for _, d := range docs {
		if err := db.Create(d).Error; err != nil {
			t.Fatal(err)
		}
	}
	stores := []*model.Storage{
		{DocumentID: docs[0].ID, Provider: "s3", FileName: "a.pdf", Status: "stored", SizeBytes: 100},
		{DocumentID: docs[1].ID, Provider: "s3", FileName: "b.pdf", Status: "stored", SizeBytes: 200},
		{DocumentID: docs[2].ID, Provider: "local", FileName: "c.pdf", Status: "stored", SizeBytes: 50},
	}
	for _, s := range stores {
		if err := db.Create(s).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.Verification{DocumentID: docs[1].ID, RequestedBy: owner, Status: constant.StatusPending, Method: "manual"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Approval{DocumentID: docs[2].ID, Level: 1, ApproverID: owner, RequestedBy: owner, Status: constant.StatusPending}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestAnalyticsDocuments(t *testing.T) {
	svc, db := analyticsHarness(t)
	seedAnalyticsData(t, db)

	out, err := svc.Documents(context.Background(), 14)
	if err != nil {
		t.Fatal(err)
	}
	byStatus := out["by_status"].(map[string]int64)
	if byStatus[constant.DocDraft] != 2 {
		t.Errorf("draft count = %d, want 2", byStatus[constant.DocDraft])
	}
	byCategory := out["by_category"].(map[string]int64)
	if byCategory["Legal"] != 2 || byCategory["HR"] != 1 || byCategory["uncategorized"] != 1 {
		t.Errorf("by_category = %v", byCategory)
	}
	trend := out["documents_per_day"].(map[string]int64)
	total := int64(0)
	for _, c := range trend {
		total += c
	}
	if total != 4 {
		t.Errorf("documents_per_day totals %d, want 4", total)
	}
}

func TestAnalyticsStorage(t *testing.T) {
	svc, db := analyticsHarness(t)
	seedAnalyticsData(t, db)

	out, err := svc.Storage(context.Background(), 14)
	if err != nil {
		t.Fatal(err)
	}
	if out["total_bytes"].(int64) != 350 {
		t.Errorf("total_bytes = %v, want 350", out["total_bytes"])
	}
	byProvider := out["by_provider"].(map[string]int64)
	if byProvider["s3"] != 300 || byProvider["local"] != 50 {
		t.Errorf("by_provider = %v", byProvider)
	}
	trend := out["bytes_per_day"].(map[string]int64)
	total := int64(0)
	for _, b := range trend {
		total += b
	}
	if total != 350 {
		t.Errorf("bytes_per_day totals %d, want 350", total)
	}
}

func TestAnalyticsWorkflow(t *testing.T) {
	svc, db := analyticsHarness(t)
	seedAnalyticsData(t, db)

	out, err := svc.Workflow(context.Background(), 14)
	if err != nil {
		t.Fatal(err)
	}
	funnel := out["funnel"].(map[string]int64)
	for _, st := range []string{constant.DocDraft, constant.DocPendingVerif, constant.DocVerified,
		constant.DocRejected, constant.DocApproved, constant.DocArchived} {
		if _, ok := funnel[st]; !ok {
			t.Errorf("funnel missing status %q: %v", st, funnel)
		}
	}
	if funnel[constant.DocApproved] != 1 {
		t.Errorf("approved = %d, want 1", funnel[constant.DocApproved])
	}
	if out["pending_verifications"].(int64) != 1 {
		t.Errorf("pending_verifications = %v, want 1", out["pending_verifications"])
	}
	if out["pending_approvals"].(int64) != 1 {
		t.Errorf("pending_approvals = %v, want 1", out["pending_approvals"])
	}
}
