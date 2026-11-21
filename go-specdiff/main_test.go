package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustLoad(t *testing.T, raw string) snapshot {
	t.Helper()
	s, err := load(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDiff(t *testing.T) {
	old := mustLoad(t, `{"revision":"r1","items":[{"id":"d","unit":"m","quantity":7},{"id":"c","unit":"pc","quantity":2},{"id":"b","unit":"pc","quantity":11},{"id":"e","unit":"pc","quantity":5}]}`)
	next := mustLoad(t, `{"revision":"r2","items":[{"id":"d","unit":"cm","quantity":700},{"id":"a","unit":"pc","quantity":0},{"id":"b","unit":"pc","quantity":4},{"id":"e","unit":"pc","quantity":5}]}`)
	var got bytes.Buffer
	diff(old, next, &got)
	want := `Revision "r1" -> "r2"
ADDED "a": 0 "pc"
QUANTITY CHANGED "b": 11 -> 4 "pc" (-7)
REMOVED "c": 2 "pc"
UNIT CHANGED "d": 7 "m" -> 700 "cm" (no numeric comparison)
`
	if got.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got.String(), want)
	}
	got.Reset()
	diff(old, old, &got)
	if !strings.Contains(got.String(), "No item changes.") {
		t.Fatal(got.String())
	}
}

func TestInvalidSnapshots(t *testing.T) {
	for _, raw := range []string{`null`, `{"revision":"r"}`, `{"revision":"r","items":null}`, `{"revision":"r","items":[]} {}`, `{"revision":"r","items":[],"unknown":0}`, `{"revision":"r","items":[{"id":"a","unit":"pc"}]}`, `{"revision":"r","items":[{"id":"a","unit":"pc","quantity":-1}]}`, `{"revision":"r","items":[{"id":"a","unit":"pc","quantity":1.5}]}`, `{"revision":"r","items":[{"id":"a","unit":"pc","quantity":1},{"id":"a","unit":"m","quantity":3}]}`} {
		if _, err := load(strings.NewReader(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestCLI(t *testing.T) {
	p := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(p, []byte(strings.Repeat(" ", (1<<20)+1)), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		code int
	}{
		{nil, 2},
		{[]string{"--help"}, 0},
		{[]string{"examples/old.json", "examples/new.json"}, 0},
		{[]string{"examples/old.json", "missing.json"}, 1},
		{[]string{p, "examples/new.json"}, 1},
	} {
		var out, diagnostics bytes.Buffer
		if got := run(tc.args, &out, &diagnostics); got != tc.code {
			t.Errorf("%v: got %d, want %d (%s)", tc.args, got, tc.code, diagnostics.String())
		}
	}
}
