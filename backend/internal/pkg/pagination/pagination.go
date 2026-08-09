// Package pagination implements server-side cursor (keyset) pagination.
// Cursors are opaque base64 tokens encoding the last row's sort value and id,
// which keeps pagination stable across inserts and allows dynamic sorting.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	// DefaultLimit is applied when the client omits a limit.
	DefaultLimit = 20
	// MaxLimit caps page size to protect the server.
	MaxLimit = 100
)

// ErrInvalidCursor is returned for malformed cursor tokens.
var ErrInvalidCursor = errors.New("invalid cursor")

// Request is the pagination envelope embedded in every list request body.
// Filters, sort and date-range are all transported in the body (no query
// params, per project convention).
type Request struct {
	Cursor   string         `json:"cursor"`
	Limit    int            `json:"limit"`
	Filters  map[string]any `json:"filters"`
	SortBy   string         `json:"sort_by"`
	SortDir  string         `json:"sort_dir"`
	DateFrom string         `json:"date_from"`
	DateTo   string         `json:"date_to"`
	Search   string         `json:"search"`
}

// Normalized is a validated, defaulted view of Request.
type Normalized struct {
	Cursor   string
	Limit    int
	Filters  map[string]any
	SortBy   string
	SortDir  string
	DateFrom *time.Time
	DateTo   *time.Time
	Search   string
}

// TimeLayouts accepted for date filters (RFC3339 or plain dates).
var TimeLayouts = []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05"}

// Normalize validates and defaults a Request.
func (r *Request) Normalize(defaultSort string) (*Normalized, error) {
	n := &Normalized{
		Cursor:  strings.TrimSpace(r.Cursor),
		Limit:   r.Limit,
		Filters: r.Filters,
		SortBy:  strings.TrimSpace(r.SortBy),
		SortDir: strings.ToLower(strings.TrimSpace(r.SortDir)),
		Search:  strings.TrimSpace(r.Search),
	}
	if n.Limit <= 0 {
		n.Limit = DefaultLimit
	}
	if n.Limit > MaxLimit {
		n.Limit = MaxLimit
	}
	if n.SortBy == "" {
		n.SortBy = defaultSort
	}
	if n.SortDir == "" {
		n.SortDir = "desc"
	}
	if n.SortDir != "asc" && n.SortDir != "desc" {
		return nil, errors.New("sort_dir must be asc or desc")
	}
	if r.DateFrom != "" {
		t, err := ParseTime(r.DateFrom)
		if err != nil {
			return nil, errors.New("date_from is invalid")
		}
		n.DateFrom = &t
	}
	if r.DateTo != "" {
		t, err := ParseTime(r.DateTo)
		if err != nil {
			return nil, errors.New("date_to is invalid")
		}
		n.DateTo = &t
	}
	return n, nil
}

// ParseTime parses a date filter into a time.Time.
func ParseTime(s string) (time.Time, error) {
	for _, layout := range TimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}

// CursorValue is the decoded contents of a cursor token.
type CursorValue struct {
	Kind string     `json:"k"` // string | number | time
	Str  string     `json:"s,omitempty"`
	Num  float64    `json:"n,omitempty"`
	Time *time.Time `json:"t,omitempty"`
	ID   string     `json:"i"`
}

// Value returns the typed sort value of the cursor.
func (c *CursorValue) Value() any {
	switch c.Kind {
	case "number":
		return c.Num
	case "time":
		if c.Time != nil {
			return *c.Time
		}
	}
	return c.Str
}

// EncodeCursor builds an opaque cursor from a sort value and last id.
func EncodeCursor(sortValue any, id string) string {
	cv := CursorValue{ID: id}
	switch v := sortValue.(type) {
	case time.Time:
		cv.Kind = "time"
		cv.Time = &v
	case float64, float32, int, int64, int32, uint, uint64:
		cv.Kind = "number"
		switch n := v.(type) {
		case float64:
			cv.Num = n
		case float32:
			cv.Num = float64(n)
		case int:
			cv.Num = float64(n)
		case int64:
			cv.Num = float64(n)
		case int32:
			cv.Num = float64(n)
		case uint:
			cv.Num = float64(n)
		case uint64:
			cv.Num = float64(n)
		}
	default:
		cv.Kind = "string"
		cv.Str = toString(v)
	}
	raw, err := json.Marshal(cv)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor parses an opaque cursor back into its typed value.
func DecodeCursor(s string) (*CursorValue, error) {
	if s == "" {
		return nil, ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var cv CursorValue
	if err := json.Unmarshal(raw, &cv); err != nil || cv.ID == "" {
		return nil, ErrInvalidCursor
	}
	return &cv, nil
}

// Meta is the pagination summary returned on every list.
type Meta struct {
	NextCursor    string `json:"next_cursor"`
	HasMore       bool   `json:"has_more"`
	Limit         int    `json:"limit"`
	ReturnedCount int    `json:"returned_count"`
	TotalCount    int64  `json:"total_count"`
}

// BuildMeta fills a Meta given the fetched (limit+1) rows.
func BuildMeta(rows int, limit int, total int64, lastSortValue any, lastID string) Meta {
	meta := Meta{
		Limit:      limit,
		TotalCount: total,
	}
	if rows > limit {
		meta.HasMore = true
		meta.ReturnedCount = limit
		meta.NextCursor = EncodeCursor(lastSortValue, lastID)
	} else {
		meta.HasMore = false
		meta.ReturnedCount = rows
	}
	return meta
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(raw)
}
