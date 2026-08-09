package service

import (
	"context"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"gorm.io/gorm"
)

// Option is the universal {id, name} shape dropdowns consume.
type Option struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// optionKinds is the whitelist of entity types the options endpoint serves.
// The label column differs per entity (users/roles/categories/templates use
// "name", documents use "title"); searchCols lists columns searched for the
// substring. Users are restricted to active accounts so dropdowns never offer
// revoked/locked users for assignment.
var optionKinds = map[string]struct {
	model      any
	nameCol    string
	searchCols []string
	extraCond  string
}{
	"users":      {model: &model.User{}, nameCol: "name", searchCols: []string{"name", "email"}, extraCond: "status = 'active'"},
	"roles":      {model: &model.Role{}, nameCol: "name", searchCols: []string{"name"}},
	"categories": {model: &model.Category{}, nameCol: "name", searchCols: []string{"name"}},
	"templates":  {model: &model.Template{}, nameCol: "name", searchCols: []string{"name"}},
	"documents":  {model: &model.Document{}, nameCol: "title", searchCols: []string{"title"}},
}

// OptionsService provides id+name lookups for form dropdowns.
type OptionsService struct {
	db *gorm.DB
}

// NewOptionsService wires the lookup service.
func NewOptionsService(db *gorm.DB) *OptionsService { return &OptionsService{db: db} }

// List returns up to limit {id, name} pairs for kind, filtered by search
// (case-insensitive substring). Users are restricted to active accounts so
// dropdowns never offer revoked/locked users for assignment.
func (s *OptionsService) List(ctx context.Context, kind, search string, limit int) ([]Option, error) {
	spec, ok := optionKinds[kind]
	if !ok {
		return nil, apperror.BadRequest("unknown option type: " + kind)
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	tx := s.db.WithContext(ctx).
		Model(spec.model).
		Select("id", spec.nameCol+" AS name").
		Order(spec.nameCol + " ASC").
		Limit(limit)
	if spec.extraCond != "" {
		tx = tx.Where(spec.extraCond)
	}
	if search = strings.TrimSpace(search); search != "" {
		// Escape LIKE wildcards so a literal % or _ in user input never acts as
		// a pattern (a search for "100%" must not match every row).
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(search)
		parts := make([]string, 0, len(spec.searchCols))
		args := make([]any, 0, len(spec.searchCols))
		for _, col := range spec.searchCols {
			parts = append(parts, "LOWER("+col+") LIKE LOWER(?) ESCAPE '\\'")
			args = append(args, "%"+escaped+"%")
		}
		tx = tx.Where("("+strings.Join(parts, " OR ")+")", args...)
	}

	var options []Option
	if err := tx.Scan(&options).Error; err != nil {
		return nil, apperror.Internal("failed to load options", err)
	}
	return options, nil
}
