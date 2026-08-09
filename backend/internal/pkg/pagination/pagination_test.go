package pagination

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	ts := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		value any
		id    string
	}{
		{"string", "abc", "uuid-1"},
		{"time", ts, "uuid-2"},
		{"int", 42, "uuid-3"},
		{"float", 3.14, "uuid-4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc := EncodeCursor(tc.value, tc.id)
			if enc == "" {
				t.Fatal("empty cursor")
			}
			dec, err := DecodeCursor(enc)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if dec.ID != tc.id {
				t.Errorf("id mismatch: got %s want %s", dec.ID, tc.id)
			}
			switch v := dec.Value().(type) {
			case time.Time:
				if !v.Equal(ts) {
					t.Errorf("time mismatch: %v", v)
				}
			case float64:
				if v != 42 && v != 3.14 {
					t.Errorf("number mismatch: %v", v)
				}
			case string:
				if v != "abc" {
					t.Errorf("string mismatch: %v", v)
				}
			default:
				t.Errorf("unexpected type %T", v)
			}
		})
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	for _, s := range []string{"", "%%%", "aGVsbG8="} {
		if _, err := DecodeCursor(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestNormalizeDefaults(t *testing.T) {
	r := &Request{Limit: 0, SortDir: "", SortBy: ""}
	n, err := r.Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	if n.Limit != DefaultLimit {
		t.Errorf("limit = %d", n.Limit)
	}
	if n.SortBy != "created_at" || n.SortDir != "desc" {
		t.Errorf("sort defaults wrong: %s %s", n.SortBy, n.SortDir)
	}
}

func TestNormalizeCapsLimitAndRejectsBadDir(t *testing.T) {
	r := &Request{Limit: 10000}
	n, _ := r.Normalize("created_at")
	if n.Limit != MaxLimit {
		t.Errorf("limit not capped: %d", n.Limit)
	}
	r2 := &Request{SortDir: "sideways"}
	if _, err := r2.Normalize("created_at"); err == nil {
		t.Error("expected error for bad sort_dir")
	}
}

func TestNormalizeDateRange(t *testing.T) {
	r := &Request{DateFrom: "2026-01-01", DateTo: "2026-01-02T10:00:00Z"}
	n, err := r.Normalize("created_at")
	if err != nil {
		t.Fatal(err)
	}
	if n.DateFrom == nil || n.DateTo == nil {
		t.Fatal("dates not parsed")
	}
	r2 := &Request{DateFrom: "not-a-date"}
	if _, err := r2.Normalize("created_at"); err == nil {
		t.Error("expected error for bad date")
	}
}

func TestBuildMeta(t *testing.T) {
	meta := BuildMeta(21, 20, 100, time.Now(), "last-id")
	if !meta.HasMore || meta.NextCursor == "" || meta.ReturnedCount != 20 {
		t.Errorf("has-more meta wrong: %+v", meta)
	}
	meta2 := BuildMeta(5, 20, 100, nil, "")
	if meta2.HasMore || meta2.NextCursor != "" || meta2.ReturnedCount != 5 {
		t.Errorf("last-page meta wrong: %+v", meta2)
	}
}
