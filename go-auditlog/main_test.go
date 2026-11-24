package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendReadAndInvalidInputDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	var out, diag bytes.Buffer
	now := func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC+1", 3600)) }
	call := func(code int, args ...string) {
		t.Helper()
		if n := run(args, &out, &diag, now); n != code {
			t.Fatalf("got %d, want %d: %s", n, code, diag.String())
		}
	}
	call(0, "append", "-file", path, "-action", "created", "-subject", "demo-1")
	call(2, "append", "-file", path, "-action", "bad")
	call(0, "append", "-file", path, "-action", "closed", "-subject", "demo-2\npart")
	call(0, "read", "-file", path)
	want := `{"time":"2026-01-02T02:04:05Z","action":"created","subject":"demo-1"}
{"time":"2026-01-02T02:04:05Z","action":"closed","subject":"demo-2\npart"}
`
	file, err := os.ReadFile(path)
	if err != nil || string(file) != want || out.String() != want {
		t.Fatalf("file=%q; out=%q; error=%v", file, out.String(), err)
	}
}

func TestReadReportsCorruptRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte(`{"time":"2026-01-02T02:04:05Z","action":"created","subject":"demo-1"}
broken
`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if code := run([]string{"read", "-file", path}, &out, &diag, time.Now); code != 1 ||
		!strings.Contains(diag.String(), "line 2") || strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("code=%d out=%q diag=%q", code, out.String(), diag.String())
	}
}

func TestMissingFile(t *testing.T) {
	var out, diag bytes.Buffer
	if code := run([]string{"read", "-file", filepath.Join(t.TempDir(), "missing")}, &out, &diag, time.Now); code != 1 || diag.Len() == 0 {
		t.Fatalf("code=%d diag=%q", code, diag.String())
	}
}
