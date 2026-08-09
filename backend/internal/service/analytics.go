package service

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"gorm.io/gorm"
)

// AnalyticsService is the README analytics module (AnalyticsService):
// cross-sectional and time-series aggregates over documents, storage usage
// and the verification / approval workflows.
type AnalyticsService interface {
	Documents(ctx context.Context, days int) (map[string]any, error)
	Storage(ctx context.Context, days int) (map[string]any, error)
	Workflow(ctx context.Context, days int) (map[string]any, error)
}

// analyticsService implements AnalyticsService using portable GORM queries
// (no SQL date functions — grouping happens in Go, so the same code runs on
// Postgres and SQLite).
type analyticsService struct {
	db *gorm.DB
}

// NewAnalyticsService wires the analytics domain.
func NewAnalyticsService(db *gorm.DB) AnalyticsService {
	return &analyticsService{db: db}
}

// Documents aggregates documents by status, by category and per-day creation
// within the window (days defaults to 14, capped at 90).
func (s *analyticsService) Documents(ctx context.Context, days int) (map[string]any, error) {
	out := map[string]any{}
	byStatus, err := s.countGrouped(ctx, &model.Document{}, "status", nil)
	if err != nil {
		return nil, err
	}
	out["by_status"] = byStatus
	byCategory, err := s.documentsByCategory(ctx)
	if err != nil {
		return nil, err
	}
	out["by_category"] = byCategory
	trend, err := s.dailyCount(ctx, days, &model.Document{})
	if err != nil {
		return nil, err
	}
	out["documents_per_day"] = trend
	return out, nil
}

// Storage aggregates total bytes, per-provider bytes and the per-day upload
// trend within the window.
func (s *analyticsService) Storage(ctx context.Context, days int) (map[string]any, error) {
	var total struct {
		Sum int64 `gorm:"column:sum"`
	}
	if err := s.db.WithContext(ctx).Model(&model.Storage{}).
		Select("COALESCE(SUM(size_bytes),0) AS sum").Scan(&total).Error; err != nil {
		return nil, err
	}
	out := map[string]any{"total_bytes": total.Sum}

	rows := []struct {
		Key string `gorm:"column:key"`
		Sum int64  `gorm:"column:sum"`
	}{}
	if err := s.db.WithContext(ctx).Model(&model.Storage{}).
		Select("provider AS key, COALESCE(SUM(size_bytes),0) AS sum").
		Group("provider").Scan(&rows).Error; err != nil {
		return nil, err
	}
	byProvider := map[string]int64{}
	for _, r := range rows {
		byProvider[r.Key] = r.Sum
	}
	out["by_provider"] = byProvider

	trend, err := s.dailyBytes(ctx, days)
	if err != nil {
		return nil, err
	}
	out["bytes_per_day"] = trend
	return out, nil
}

// Workflow returns the document status funnel (with every known status
// present, zero-filled) and the pending verification / approval backlogs.
func (s *analyticsService) Workflow(ctx context.Context, days int) (map[string]any, error) {
	funnel, err := s.countGrouped(ctx, &model.Document{}, "status", nil)
	if err != nil {
		return nil, err
	}
	for _, st := range []string{constant.DocDraft, constant.DocPendingVerif, constant.DocVerified,
		constant.DocRejected, constant.DocApproved, constant.DocArchived} {
		if _, ok := funnel[st]; !ok {
			funnel[st] = 0
		}
	}
	out := map[string]any{"funnel": funnel}

	var pendingVer int64
	if err := s.db.WithContext(ctx).Model(&model.Verification{}).
		Where("status = ?", constant.StatusPending).Count(&pendingVer).Error; err != nil {
		return nil, err
	}
	out["pending_verifications"] = pendingVer

	var pendingApp int64
	if err := s.db.WithContext(ctx).Model(&model.Approval{}).
		Where("status = ?", constant.StatusPending).Count(&pendingApp).Error; err != nil {
		return nil, err
	}
	out["pending_approvals"] = pendingApp
	return out, nil
}

// countGrouped returns {key: count} grouped by a column of the given model,
// optionally restricted to rows created after since.
func (s *analyticsService) countGrouped(ctx context.Context, m any, column string, since *time.Time) (map[string]int64, error) {
	tx := s.db.WithContext(ctx).Model(m)
	if since != nil {
		tx = tx.Where("created_at >= ?", *since)
	}
	rows := []struct {
		Key string `gorm:"column:key"`
		Cnt int64  `gorm:"column:cnt"`
	}{}
	if err := tx.Select(column + " AS key, COUNT(*) AS cnt").Group(column).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.Key] = r.Cnt
	}
	return out, nil
}

// documentsByCategory counts documents per category name; documents without a
// category are grouped under "uncategorized".
func (s *analyticsService) documentsByCategory(ctx context.Context) (map[string]int64, error) {
	rows := []struct {
		Key *string `gorm:"column:key"`
		Cnt int64   `gorm:"column:cnt"`
	}{}
	if err := s.db.WithContext(ctx).Model(&model.Document{}).
		Select("categories.name AS key, COUNT(*) AS cnt").
		Joins("LEFT JOIN categories ON categories.id = documents.category_id").
		Group("categories.name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		k := "uncategorized"
		if r.Key != nil {
			k = *r.Key
		}
		out[k] = r.Cnt
	}
	return out, nil
}

// dailyCount buckets rows of m by calendar day within the window.
func (s *analyticsService) dailyCount(ctx context.Context, days int, m any) (map[string]int64, error) {
	since := sinceDays(days)
	rows := []struct {
		CreatedAt time.Time `gorm:"column:created_at"`
	}{}
	if err := s.db.WithContext(ctx).Model(m).Where("created_at >= ?", since).
		Select("created_at").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.CreatedAt.Format("2006-01-02")]++
	}
	return out, nil
}

// dailyBytes sums storage bytes by calendar day within the window.
func (s *analyticsService) dailyBytes(ctx context.Context, days int) (map[string]int64, error) {
	since := sinceDays(days)
	rows := []struct {
		CreatedAt time.Time `gorm:"column:created_at"`
		SizeBytes int64     `gorm:"column:size_bytes"`
	}{}
	if err := s.db.WithContext(ctx).Model(&model.Storage{}).Where("created_at >= ?", since).
		Select("created_at, size_bytes").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.CreatedAt.Format("2006-01-02")] += r.SizeBytes
	}
	return out, nil
}

// sinceDays normalizes the analytics window (14 default, 90 max).
func sinceDays(days int) time.Time {
	if days <= 0 || days > 90 {
		days = 14
	}
	return time.Now().AddDate(0, 0, -days)
}

var _ AnalyticsService = (*analyticsService)(nil)
