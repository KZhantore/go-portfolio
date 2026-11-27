package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const bodyLimit = 4096

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type request struct {
	Item  string `json:"item"`
	Units int    `json:"units"`
}

type order struct {
	ID int `json:"id"`
	request
}

type ledger struct {
	mu       sync.Mutex
	entries  map[string]order
	capacity int
}

var (
	errConflict = errors.New("key already used with a different order")
	errCapacity = errors.New("demo ledger is full")
)

func (l *ledger) create(key string, req request) (order, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if old, found := l.entries[key]; found {
		if old.request != req {
			return order{}, false, errConflict
		}
		return old, true, nil
	}
	if len(l.entries) >= l.capacity {
		return order{}, false, errCapacity
	}
	result := order{ID: len(l.entries) + 1, request: req}
	l.entries[key] = result
	return result, false, nil
}

func handler(capacity int) http.Handler {
	l := &ledger{entries: make(map[string]order), capacity: capacity}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		keys := r.Header.Values("Idempotency-Key")
		if len(keys) != 1 || !keyPattern.MatchString(keys[0]) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provide one valid Idempotency-Key"})
			return
		}
		defer r.Body.Close()
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyLimit))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "order body exceeds 4096 bytes"})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read order body"})
			return
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		var req request
		if err := dec.Decode(&req); err != nil || dec.Decode(new(any)) != io.EOF {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected one JSON order"})
			return
		}
		req.Item = strings.TrimSpace(req.Item)
		if len(req.Item) < 1 || len(req.Item) > 80 || req.Units < 1 || req.Units > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "item must be 1–80 bytes and units must be 1–100"})
			return
		}

		result, replay, err := l.create(keys[0], req)
		switch err {
		case errConflict:
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		case errCapacity:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		if replay {
			w.Header().Set("Idempotency-Replayed", "true")
		}
		writeJSON(w, http.StatusCreated, result)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	srv := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           handler(1000),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Print("demo listening at http://127.0.0.1:8080")
	log.Fatal(srv.ListenAndServe())
}
