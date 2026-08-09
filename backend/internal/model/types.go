package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// NullableString returns a pointer to s, or nil for an empty string. Used to
// store optional UUID foreign keys as NULL instead of an invalid "" value.
func NullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// JSONMap stores an arbitrary object in a jsonb column.
type JSONMap map[string]any

// Value implements driver.Valuer.
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner.
func (j *JSONMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONMap")
	}
	return json.Unmarshal(b, j)
}

// JSONArray stores a string slice in a jsonb column.
type JSONArray []string

// Value implements driver.Valuer.
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner.
func (j *JSONArray) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONArray")
	}
	return json.Unmarshal(b, j)
}
