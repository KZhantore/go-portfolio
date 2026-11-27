package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func post(h http.Handler, key, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(body))
	r.Header.Set("Idempotency-Key", key)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestReplayConflictAndCapacity(t *testing.T) {
	h := handler(1)
	first := post(h, "sample-1", `{"item":"notebook","units":2}`)
	replay := post(h, "sample-1", `{ "units":2, "item":" notebook " }`)
	conflict := post(h, "sample-1", `{"item":"notebook","units":3}`)
	full := post(h, "sample-2", `{"item":"notebook","units":2}`)
	if first.Code != 201 || replay.Code != 201 || replay.Body.String() != first.Body.String() ||
		first.Header().Get("Idempotency-Replayed") != "" || replay.Header().Get("Idempotency-Replayed") != "true" ||
		conflict.Code != 409 || full.Code != 503 {
		t.Fatalf("first=%v replay=%v conflict=%v full=%v", first, replay, conflict, full)
	}
}

func TestValidationDoesNotConsumeKey(t *testing.T) {
	h := handler(1)
	for _, tc := range []struct {
		key, body string
		status    int
	}{
		{"", `{"item":"notebook","units":2}`, 400},
		{"spaces here", `{"item":"notebook","units":2}`, 400},
		{"sample-1", `{"item":"notebook","units":1,"other":true}`, 400},
		{"sample-1", `{"item":"notebook","units":1} {}`, 400},
		{"sample-1", `{"item":" ","units":1}`, 400},
		{"sample-1", `{"item":"notebook","units":101}`, 400},
		{"sample-1", strings.Repeat(" ", bodyLimit+1), 413},
		{"sample-1", `{"item":"notebook","units":2}`, 201},
	} {
		w := post(h, tc.key, tc.body)
		if w.Code != tc.status {
			t.Errorf("key=%q body length=%d: got %d want %d", tc.key, len(tc.body), w.Code, tc.status)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"item":"notebook","units":2}`))
	r.Header.Add("Idempotency-Key", "sample-1")
	r.Header.Add("Idempotency-Key", "sample-2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("multiple keys accepted: %d", w.Code)
	}
}

func TestConcurrentSameKeyCreatesOneOrder(t *testing.T) {
	h := handler(32)
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 32)
	start := make(chan struct{})
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- post(h, "sample-1", `{"item":"notebook","units":2}`)
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	firstCount := 0
	for w := range results {
		var got order
		if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &got) != nil || got.ID != 1 {
			t.Fatalf("unexpected response: %d %q", w.Code, w.Body)
		}
		if w.Header().Get("Idempotency-Replayed") == "" {
			firstCount++
		}
	}
	if firstCount != 1 {
		t.Fatalf("fresh creations=%d, want 1", firstCount)
	}
	next := post(h, "sample-2", `{"item":"notebook","units":2}`)
	var got order
	if next.Code != 201 || json.Unmarshal(next.Body.Bytes(), &got) != nil || got.ID != 2 {
		t.Fatalf("unexpected next order: %d %q", next.Code, next.Body)
	}
}

type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("broken body") }
func (brokenBody) Close() error             { return nil }

func TestReadError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/orders", brokenBody{})
	r.Header.Set("Idempotency-Key", "sample-1")
	w := httptest.NewRecorder()
	handler(1).ServeHTTP(w, r)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "cannot read order body") {
		t.Fatalf("unexpected response: %d %q", w.Code, w.Body)
	}
}
