// Package repository provides a generic GORM repository implementing server
// side cursor pagination, whitelisted dynamic sorting, exact filters, keyword
// search, date ranges and per-entity summaries. No raw SQL is used.
package repository

import (
	"context"
	"reflect"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"gorm.io/gorm"
)

// ListQuery configures one list execution.
type ListQuery struct {
	P *pagination.Normalized
	// Scopes are additional query filters applied after BaseScope (e.g.
	// row-level access control). They run for the row query, the count and
	// the entity summary so the visible page and its aggregates agree.
	Scopes []func(*gorm.DB) *gorm.DB
}

// Repo is a generic GORM repository for entity T.
type Repo[T any] struct {
	DB          *gorm.DB
	SortMap     map[string]string // client sort key -> db column (whitelist)
	FilterMap   map[string]string // filter key -> db column (whitelist)
	SearchCols  []string
	BaseScope   func(tx *gorm.DB) *gorm.DB
	SummaryFn   func(ctx context.Context, q ListQuery) (map[string]any, error)
	Preloads    []string
}

// New builds a repository for T with the given column whitelists.
func New[T any](db *gorm.DB, sortMap, filterMap map[string]string, searchCols []string) *Repo[T] {
	if sortMap == nil {
		sortMap = map[string]string{"created_at": "created_at"}
	}
	if filterMap == nil {
		filterMap = map[string]string{}
	}
	if _, ok := sortMap["id"]; !ok {
		sortMap["id"] = "id"
	}
	return &Repo[T]{DB: db, SortMap: sortMap, FilterMap: filterMap, SearchCols: searchCols}
}

// Create inserts an entity (UUID v7 assigned by the model hook).
func (r *Repo[T]) Create(ctx context.Context, m *T) error {
	return r.DB.WithContext(ctx).Create(m).Error
}

// Save persists full entity state.
func (r *Repo[T]) Save(ctx context.Context, m *T) error {
	return r.DB.WithContext(ctx).Save(m).Error
}

// GetByID loads an entity by primary key with configured preloads.
func (r *Repo[T]) GetByID(ctx context.Context, id string) (*T, error) {
	var m T
	tx := r.DB.WithContext(ctx)
	for _, p := range r.Preloads {
		tx = tx.Preload(p)
	}
	err := tx.First(&m, "id = ?", id).Error
	if err != nil {
		if isNotFound(err) {
			return nil, apperror.NotFound(typeName[T]())
		}
		return nil, err
	}
	return &m, nil
}

// Delete soft-deletes an entity by id.
func (r *Repo[T]) Delete(ctx context.Context, id string) error {
	res := r.DB.WithContext(ctx).Where("id = ?", id).Delete(new(T))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.NotFound(typeName[T]())
	}
	return nil
}

// List runs a cursor-paginated query with filters, search, date range and
// dynamic sorting, returning rows, page meta and an optional summary. The
// configured preloads (e.g. Roles, Permissions) are applied so list rows
// carry the same relations as GetByID.
func (r *Repo[T]) List(ctx context.Context, q ListQuery) ([]T, *pagination.Meta, map[string]any, error) {
	tx := r.baseQuery(ctx, q)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, nil, nil, err
	}

	cursorTx, cursorErr := r.applyCursor(tx, q)
	if cursorErr != nil {
		return nil, nil, nil, cursorErr
	}
	tx = cursorTx
	sortCol := r.resolveSort(q)

	for _, p := range r.Preloads {
		tx = tx.Preload(p)
	}

	rows := make([]T, 0, q.P.Limit+1)
	tx = tx.Order(sortCol + " " + q.P.SortDir).Order("id " + q.P.SortDir).Limit(q.P.Limit + 1)
	if err := tx.Find(&rows).Error; err != nil {
		return nil, nil, nil, err
	}

	meta := r.buildMeta(rows, q, sortCol)
	meta.TotalCount = total
	if meta.HasMore {
		rows = rows[:q.P.Limit] // drop the look-ahead row before returning
	}
	summary, err := r.buildSummary(ctx, q)
	if err != nil {
		return nil, nil, nil, err
	}
	return rows, meta, summary, nil
}

// FirstWhere loads the first row matching a column equality filter.
func (r *Repo[T]) FirstWhere(ctx context.Context, column string, value any) (*T, error) {
	var m T
	tx := r.DB.WithContext(ctx)
	for _, p := range r.Preloads {
		tx = tx.Preload(p)
	}
	err := tx.First(&m, column+" = ?", value).Error
	if err != nil {
		if isNotFound(err) {
			return nil, apperror.NotFound(typeName[T]())
		}
		return nil, err
	}
	return &m, nil
}

// Count returns the total rows matching the base scope and filters.
func (r *Repo[T]) Count(ctx context.Context, q ListQuery) (int64, error) {
	tx := r.baseQuery(ctx, q)
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

func (r *Repo[T]) baseQuery(ctx context.Context, q ListQuery) *gorm.DB {
	tx := r.DB.WithContext(ctx).Model(new(T))
	if r.BaseScope != nil {
		tx = r.BaseScope(tx)
	}
	for _, sc := range q.Scopes {
		tx = sc(tx)
	}
	if q.P.DateFrom != nil {
		tx = tx.Where("created_at >= ?", q.P.DateFrom)
	}
	if q.P.DateTo != nil {
		tx = tx.Where("created_at <= ?", q.P.DateTo)
	}
	for k, v := range q.P.Filters {
		col, ok := r.FilterMap[k]
		if !ok {
			continue
		}
		tx = tx.Where(col+" = ?", v)
	}
	if q.P.Search != "" && len(r.SearchCols) > 0 {
		parts := make([]string, 0, len(r.SearchCols))
		args := make([]any, 0, len(r.SearchCols))
		for _, col := range r.SearchCols {
			parts = append(parts, "LOWER("+col+") LIKE LOWER(?)")
			args = append(args, "%"+q.P.Search+"%")
		}
		tx = tx.Where("("+strings.Join(parts, " OR ")+")", args...)
	}
	return tx
}

func (r *Repo[T]) applyCursor(tx *gorm.DB, q ListQuery) (*gorm.DB, error) {
	if q.P.Cursor == "" {
		return tx, nil
	}
	cv, err := pagination.DecodeCursor(q.P.Cursor)
	if err != nil {
		return nil, apperror.BadRequest("invalid cursor")
	}
	sortCol := r.resolveSort(q)
	if sortCol == "id" {
		return tx.Where("id "+q.P.SortDir+" ?", cv.ID), nil
	}
	if q.P.SortDir == "desc" {
		return tx.Where("("+sortCol+" < ? OR ("+sortCol+" = ? AND id < ?))",
			cv.Value(), cv.Value(), cv.ID), nil
	}
	return tx.Where("("+sortCol+" > ? OR ("+sortCol+" = ? AND id > ?))",
		cv.Value(), cv.Value(), cv.ID), nil
}

func (r *Repo[T]) resolveSort(q ListQuery) string {
	if col, ok := r.SortMap[q.P.SortBy]; ok {
		return col
	}
	return "created_at"
}

func (r *Repo[T]) buildMeta(rows []T, q ListQuery, sortCol string) *pagination.Meta {
	hasMore := len(rows) > q.P.Limit
	meta := &pagination.Meta{Limit: q.P.Limit, TotalCount: 0}
	if hasMore {
		meta.HasMore = true
		meta.ReturnedCount = q.P.Limit
		last := rows[q.P.Limit-1]
		id, _ := ColumnValue(last, "id")
		sv, _ := ColumnValue(last, sortCol)
		meta.NextCursor = pagination.EncodeCursor(sv, id.(string))
	} else {
		meta.HasMore = false
		meta.ReturnedCount = len(rows)
	}
	return meta
}

// buildSummary runs the entity-specific aggregation hook, if any.
func (r *Repo[T]) buildSummary(ctx context.Context, q ListQuery) (map[string]any, error) {
	if r.SummaryFn != nil {
		return r.SummaryFn(ctx, q)
	}
	return nil, nil
}

// ColumnValue reflects the value of a struct field by its gorm column name.
func ColumnValue(v any, column string) (any, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	return findColumn(rv, column)
}

func findColumn(rv reflect.Value, column string) (any, bool) {
	t := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		f := t.Field(i)
		fv := rv.Field(i)
		if f.Anonymous {
			if v, ok := findColumn(fv, column); ok {
				return v, true
			}
			continue
		}
		if strings.Contains(f.Tag.Get("gorm"), "column:"+column) {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return nil, false
				}
				return fv.Elem().Interface(), true
			}
			return fv.Interface(), true
		}
	}
	return nil, false
}

func isNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func typeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return "record"
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}
