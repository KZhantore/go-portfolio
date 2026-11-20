package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOrderStatusesAndNoRedirect(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("expected GET")
		}
		switch r.URL.Path {
		case "/ok":
			time.Sleep(20 * time.Millisecond)
			w.WriteHeader(204)
		case "/redirect":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/target":
			hits.Add(1)
		default:
			w.WriteHeader(503)
		}
	}))
	defer server.Close()
	var out, diagnostics bytes.Buffer
	code := run([]string{"-workers=3", server.URL + "/ok", server.URL + "/redirect", server.URL + "/bad"}, &out, &diagnostics)
	want := fmt.Sprintf("OK %q HTTP 204\nFAIL %q HTTP 302\nFAIL %q HTTP 503\n", server.URL+"/ok", server.URL+"/redirect", server.URL+"/bad")
	if code != 1 || out.String() != want || hits.Load() != 0 {
		t.Fatalf("code=%d output=%q redirect hits=%d diagnostics=%s", code, out.String(), hits.Load(), diagnostics.String())
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	}))
	defer server.Close()
	var out bytes.Buffer
	if code := run([]string{"-timeout=10ms", server.URL}, &out, &bytes.Buffer{}); code != 1 || !strings.Contains(out.String(), "timed out") {
		t.Fatalf("code=%d output=%s", code, out.String())
	}
}

func TestUsageAndSuccess(t *testing.T) {
	for _, args := range [][]string{{}, {"relative"}, {"ftp://example.com"}, {"https://user:pass@example.com"}, {"-workers=0", "https://example.com"}, {"-timeout=-1s", "https://example.com"}} {
		if code := run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
			t.Errorf("%v: code=%d", args, code)
		}
	}
	if code := run([]string{"-h"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Errorf("help code=%d", code)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer server.Close()
	if code := run([]string{server.URL}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Errorf("success code=%d", code)
	}
}

type localTransport func(*http.Request) (*http.Response, error)

func (f localTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConcurrentWorkersAreCapped(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	entered := make(chan struct{}, 6)
	release := make(chan struct{})
	client := &http.Client{Transport: localTransport(func(r *http.Request) (*http.Response, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		entered <- struct{}{}
		<-release
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	done := make(chan []result, 1)
	go func() {
		done <- checkAll(client, []string{"http://local/a", "http://local/b", "http://local/c", "http://local/d", "http://local/e", "http://local/f"}, 2)
	}()
	stillOpen := true
	defer func() {
		if stillOpen {
			close(release)
		}
	}()
	deadline := time.After(2 * time.Second)
	for i := 0; i < 6; i += 2 {
		for j := 0; j < 2; j++ {
			select {
			case <-entered:
			case <-deadline:
				t.Fatal("workers failed to run concurrently")
			}
		}
		for j := 0; j < 2; j++ {
			select {
			case release <- struct{}{}:
			case <-deadline:
				t.Fatal("could not release workers")
			}
		}
	}
	close(release)
	stillOpen = false
	select {
	case results := <-done:
		if len(results) != 6 || peak.Load() != 2 {
			t.Fatalf("results=%d peak=%d", len(results), peak.Load())
		}
	case <-deadline:
		t.Fatal("workers did not finish")
	}
}
