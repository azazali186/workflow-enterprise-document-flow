package objectstore

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeStripsUnsafeCharacters(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"report.pdf", "report.pdf"},
		{"a/b\\c..d e.png", "abc..de.png"}, // separators and spaces removed
		{"../../etc/passwd", "etcpasswd"},  // traversal stripped, no hidden file
		{".env", "env"},                    // leading dots dropped
		{"invoice#1~final!.pdf", "invoice1final.pdf"},
		{"", "file"},      // guaranteed non-empty
		{"...", "file"},   // dots-only falls back
		{"简历.pdf", "pdf"}, // non-ASCII stripped, no leading dot
		{"CON", "CON"},
	}
	for _, tc := range cases {
		if got := Sanitize(tc.in); got != tc.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeAlwaysSafe(t *testing.T) {
	for _, in := range []string{"../../x", "a/b", "C:\\evil\\file.txt", "a b\tc", "日本語", "*?<>|:\""} {
		got := Sanitize(in)
		if strings.ContainsAny(got, `/\`) || got == "" {
			t.Errorf("Sanitize(%q) = %q is unsafe", in, got)
		}
	}
}

func TestLocalStorePutAndDelete(t *testing.T) {
	dir := t.TempDir()
	store := &LocalStore{Dir: dir}

	key := NewKey("quarterly-report.xlsx")
	n, err := store.Put(t.Context(), key, strings.NewReader("hello object"), 12, "application/vnd.ms-excel")
	if err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("expected 12 bytes written, got %d", n)
	}

	full := filepath.Join(dir, key)
	raw, err := os.ReadFile(full) //nolint:gosec // G304: reading the temp-dir fixture the test just wrote
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "hello object" {
		t.Fatalf("unexpected content %q", raw)
	}

	if err := store.Delete(t.Context(), key); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(full); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err = %v", err)
	}
}

func TestLocalStoreGetRoundtrip(t *testing.T) {
	dir := t.TempDir()
	store := &LocalStore{Dir: dir}
	key := NewKey("readme.txt")
	if _, err := store.Put(t.Context(), key, strings.NewReader("payload"), 7, "text/plain"); err != nil {
		t.Fatal(err)
	}
	rc, err := store.Get(t.Context(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "payload" {
		t.Fatalf("unexpected content %q", raw)
	}
}

func TestNewKeyIsUniqueAndSafe(t *testing.T) {
	a := NewKey("a.txt")
	b := NewKey("a.txt")
	if a == b {
		t.Fatal("keys must be unique")
	}
	for _, k := range []string{a, b} {
		if strings.ContainsAny(k, `/\`) {
			t.Fatalf("key %q must not contain path separators", k)
		}
	}
}
