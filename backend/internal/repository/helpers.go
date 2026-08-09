package repository

import (
	"context"

	"gorm.io/gorm"
)

// countByStatus aggregates rows grouped by their status column.
func countByStatus(ctx context.Context, tx *gorm.DB) (map[string]any, error) {
	rows := []struct {
		Status string `gorm:"column:status"`
		Cnt    int64  `gorm:"column:cnt"`
	}{}
	if err := tx.WithContext(ctx).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]any{"total": int64(0)}
	for _, r := range rows {
		out[r.Status] = r.Cnt
		out["total"] = out["total"].(int64) + r.Cnt
	}
	return out, nil
}

// countByBool aggregates rows grouped by a boolean column (is_active, ...).
func countByBool(ctx context.Context, tx *gorm.DB, column string) (map[string]any, error) {
	rows := []struct {
		Key string `gorm:"column:key"`
		Cnt int64  `gorm:"column:cnt"`
	}{}
	if err := tx.WithContext(ctx).Select(column+" AS key, COUNT(*) AS cnt").Group(column).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]any{"total": int64(0)}
	for _, r := range rows {
		out[r.Key] = r.Cnt
		out["total"] = out["total"].(int64) + r.Cnt
	}
	return out, nil
}

// countByAction aggregates audit rows grouped by action.
func countByAction(ctx context.Context, tx *gorm.DB) (map[string]any, error) {
	rows := []struct {
		Action string `gorm:"column:action"`
		Cnt    int64  `gorm:"column:cnt"`
	}{}
	if err := tx.WithContext(ctx).Select("action, COUNT(*) AS cnt").Group("action").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]any{"total": int64(0)}
	for _, r := range rows {
		out[r.Action] = r.Cnt
		out["total"] = out["total"].(int64) + r.Cnt
	}
	return out, nil
}
