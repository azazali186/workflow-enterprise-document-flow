package service

import (
	"context"
	"reflect"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/pagination"
	"github.com/aeroxe/docu-flow/backend/internal/repository"
)

// CrudService provides reusable create/update/delete/get/list with audit for
// simple entities. T must embed model.BaseModel so GetByID and Save work.
type CrudService[T any] struct {
	Repo   *repository.Repo[T]
	Audit  *AuditService
	Entity string
}

// Create persists a new entity and audits the action.
func (s *CrudService[T]) Create(ctx context.Context, actor Actor, m *T) (*T, error) {
	if err := s.Repo.Create(ctx, m); err != nil {
		return nil, err
	}
	_ = s.Audit.Record(ctx, actor, "create", s.Entity, idOf(m), nil, m)
	return m, nil
}

// Update persists changes, capturing before/after for the audit trail. The
// incoming struct is merged over the stored row so omitted (zero-value)
// fields never wipe existing data — partial updates are safe by default.
func (s *CrudService[T]) Update(ctx context.Context, actor Actor, m *T) (*T, error) {
	id := idOf(m)
	before, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	merged := mergeForUpdate(before, m)
	if err := s.Repo.Save(ctx, merged); err != nil {
		return nil, err
	}
	_ = s.Audit.Change(ctx, actor, s.Entity, id, before, merged)
	return merged, nil
}

// Delete soft-deletes an entity after auditing the previous state.
func (s *CrudService[T]) Delete(ctx context.Context, actor Actor, id string) error {
	before, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.Audit.Record(ctx, actor, "delete", s.Entity, id, before, nil)
	return nil
}

// Get loads one entity.
func (s *CrudService[T]) Get(ctx context.Context, id string) (*T, error) {
	return s.Repo.GetByID(ctx, id)
}

// List runs the cursor-paginated listing.
func (s *CrudService[T]) List(ctx context.Context, n *pagination.Normalized) ([]T, *pagination.Meta, map[string]any, error) {
	return s.Repo.List(ctx, repository.ListQuery{P: n})
}

// mergeForUpdate keeps stored values wherever the incoming struct has zero
// values, skipping the primary key and timestamp bookkeeping fields.
func mergeForUpdate[T any](stored, incoming *T) *T {
	out := *incoming
	sv := reflect.ValueOf(stored).Elem()
	ov := reflect.ValueOf(&out).Elem()
	for i := 0; i < ov.NumField(); i++ {
		f := ov.Type().Field(i)
		fv := ov.Field(i)
		if f.Anonymous {
			mergeBase(fv, sv.Field(i))
			continue
		}
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				// Not provided → keep the stored value (incl. nil/NULL).
				if src := sv.Field(i); src.IsValid() && fv.CanSet() {
					fv.Set(src)
				}
			}
			continue
		}
		if !fv.IsZero() {
			continue
		}
		// Preserve association slices and other non-zero stored values.
		src := sv.Field(i)
		if src.IsValid() && fv.CanSet() {
			fv.Set(src)
		}
	}
	return &out
}

// mergeBase fills zero fields of the embedded BaseModel from the stored row.
func mergeBase(out, stored reflect.Value) {
	if !out.IsValid() || !stored.IsValid() || out.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < out.NumField(); i++ {
		fv := out.Field(i)
		src := stored.Field(i)
		if !fv.CanSet() || !src.IsValid() {
			continue
		}
		if fv.Kind() == reflect.Pointer || !fv.IsZero() {
			continue
		}
		fv.Set(src)
	}
}

func idOf[T any](m *T) string {
	if v, ok := any(m).(interface{ GetID() string }); ok {
		return v.GetID()
	}
	raw, _ := repository.ColumnValue(m, "id")
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}


