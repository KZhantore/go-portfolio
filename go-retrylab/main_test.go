package main

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestSimulate(t *testing.T) {
	tests := []struct {
		name, script, stop string
		p                  policy
		times              []int
	}{
		{"capped", "retry,retry,retry,retry,ok", "success", policy{5, 100, 250, 800}, []int{0, 100, 300, 550, 800}},
		{"budget", "retry,retry,ok", "wait budget exhausted", policy{5, 100, 800, 299}, []int{0, 100}},
		{"zero budget", "retry,ok", "wait budget exhausted", policy{5, 100, 800, 0}, []int{0}},
		{"limit counts first", "retry,ok", "attempt limit reached", policy{1, 100, 800, 0}, []int{0}},
		{"permanent failure", "retry,fail,ok", "permanent failure", policy{5, 100, 800, 1000}, []int{0, 100}},
		{"script exhausted", "retry", "script exhausted", policy{5, 100, 800, 1000}, []int{0}},
		{"immediate success", "ok", "success", policy{5, 100, 800, 0}, []int{0}},
		{"maximum delay", "retry,retry,ok", "wait budget exhausted", policy{5, maxMillis, maxMillis, maxMillis}, []int{0, maxMillis}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := parseScript(tt.script)
			if err != nil {
				t.Fatal(err)
			}
			trace, stop := simulate(tt.p, results)
			times := make([]int, len(trace))
			got := make([]outcome, len(trace))
			for i, a := range trace {
				times[i], got[i] = a.atMS, a.result
			}
			if stop != tt.stop || !slices.Equal(times, tt.times) || !slices.Equal(got, results[:len(tt.times)]) {
				t.Fatalf("trace=%v, stop=%q", trace, stop)
			}
		})
	}
}

func TestRun(t *testing.T) {
	for _, tt := range []struct {
		args []string
		code int
		part string
	}{
		{nil, 0, "attempt 3 at 300ms: ok\nstop: success\n"},
		{[]string{"-script", " retry , fail "}, 0, "stop: permanent failure\n"},
		{[]string{"-script", "retry,,ok"}, 2, "outcome 2"},
		{[]string{"-script", strings.Repeat("retry,", 1000) + "ok"}, 2, "1000 outcomes"},
		{[]string{"-script", strings.Repeat("x", 8193)}, 2, "8192 bytes"},
		{[]string{"-base-ms", "0"}, 2, "Expected flags only"},
		{[]string{"-attempts", "1001"}, 2, "Expected flags only"},
		{[]string{"extra"}, 2, "Expected flags only"},
		{[]string{"-bogus"}, 2, "flag provided but not defined"},
		{[]string{"-help"}, 0, "Usage"},
	} {
		var out, errOut bytes.Buffer
		got := run(tt.args, &out, &errOut)
		if got != tt.code || !strings.Contains(out.String()+errOut.String(), tt.part) {
			t.Errorf("args prefix %v: code=%d, stdout=%q, stderr=%q", len(tt.args), got, out.String(), errOut.String())
		}
	}
}

type limitedWriter struct{ writes int }

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.writes == 0 {
		return 0, io.ErrClosedPipe
	}
	w.writes--
	return len(p), nil
}

func TestOutputError(t *testing.T) {
	for _, n := range []int{0, 1} {
		if got := run([]string{"-script", "ok"}, &limitedWriter{writes: n}, io.Discard); got != 1 {
			t.Fatalf("wrote %d lines before error, code=%d; want 1", n, got)
		}
	}
}
