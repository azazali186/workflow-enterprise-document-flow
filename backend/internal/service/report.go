package service

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"gorm.io/gorm"
)

// ReportService aggregates business data for dashboards.
type ReportService interface {
	Dashboard(ctx context.Context, days int) (map[string]any, error)
}

// reportService implements ReportService using only GORM queries.
type reportService struct {
	db *gorm.DB
}

// NewReportService wires the report domain.
func NewReportService(db *gorm.DB) ReportService {
	return &reportService{db: db}
}

func (s *reportService) Dashboard(ctx context.Context, days int) (map[string]any, error) {
	if days <= 0 || days > 90 {
		days = 14
	}
	since := time.Now().AddDate(0, 0, -days)

	out := map[string]any{}

	statusCounts := func(m any, column string) (map[string]int64, error) {
		rows := []struct {
			Key string `gorm:"column:key"`
			Cnt int64  `gorm:"column:cnt"`
		}{}
		if err := s.db.WithContext(ctx).Model(m).
			Select(column + " AS key, COUNT(*) AS cnt").Group(column).Scan(&rows).Error; err != nil {
			return nil, err
		}
		res := map[string]int64{}
		for _, r := range rows {
			res[r.Key] = r.Cnt
		}
		return res, nil
	}

	var err error
	if out["documents"], err = statusCounts(&model.Document{}, "status"); err != nil {
		return nil, err
	}
	if out["verifications"], err = statusCounts(&model.Verification{}, "status"); err != nil {
		return nil, err
	}
	if out["approvals"], err = statusCounts(&model.Approval{}, "status"); err != nil {
		return nil, err
	}
	if out["storages"], err = statusCounts(&model.Storage{}, "status"); err != nil {
		return nil, err
	}
	if out["users"], err = statusCounts(&model.User{}, "status"); err != nil {
		return nil, err
	}

	var totalBytes struct {
		Sum int64 `gorm:"column:sum"`
	}
	if err := s.db.WithContext(ctx).Model(&model.Storage{}).
		Select("COALESCE(SUM(size_bytes),0) AS sum").Scan(&totalBytes).Error; err != nil {
		return nil, err
	}
	out["total_storage_bytes"] = totalBytes.Sum

	// Daily document creation counts (grouped in Go, no SQL date functions).
	type row struct {
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Model(&model.Document{}).
		Where("created_at >= ?", since).
		Select("created_at").Scan(&rows).Error; err != nil {
		return nil, err
	}
	daily := map[string]int64{}
	for _, r := range rows {
		day := r.CreatedAt.Format("2006-01-02")
		daily[day]++
	}
	out["documents_per_day"] = daily

	// Recent activity for the dashboard feed.
	var recent []model.AuditLog
	if err := s.db.WithContext(ctx).Model(&model.AuditLog{}).
		Order("created_at DESC").Limit(10).Find(&recent).Error; err != nil {
		return nil, err
	}
	out["recent_activity"] = recent

	var pending int64
	if err := s.db.WithContext(ctx).Model(&model.Approval{}).
		Where("status = ?", "pending").Count(&pending).Error; err != nil {
		return nil, err
	}
	out["pending_approvals"] = pending
	return out, nil
}

var _ ReportService = (*reportService)(nil)
