package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type input struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func answer(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	answer(w, status, map[string]string{"error": message})
}
func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotFound) {
		fail(w, 404, "task not found")
		return
	}
	log.Printf("store error: %v", err)
	fail(w, 500, "storage error")
}
func readTask(w http.ResponseWriter, r *http.Request) (input, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var fields input
	first := dec.Decode(&fields)
	var tooLarge *http.MaxBytesError
	if errors.As(first, &tooLarge) {
		fail(w, 413, "request body too large")
		return fields, false
	}
	if first != nil {
		fail(w, 400, "expected one JSON task object")
		return fields, false
	}
	extra := dec.Decode(new(any))
	if errors.As(extra, &tooLarge) {
		fail(w, 413, "request body too large")
		return fields, false
	}
	if extra != io.EOF {
		fail(w, 400, "expected one JSON task object")
		return fields, false
	}
	fields.Title = strings.TrimSpace(fields.Title)
	if n := utf8.RuneCountInString(fields.Title); n < 1 || n > 120 {
		fail(w, 400, "title must contain 1 to 120 characters")
		return fields, false
	}
	return fields, true
}

func routes(s *store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) { answer(w, 200, s.list()) })
	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		fields, ok := readTask(w, r)
		if !ok {
			return
		}
		var task Task
		err := s.change(func(next *state) error {
			task = Task{ID: next.NextID, Title: fields.Title, Done: fields.Done}
			next.NextID++
			next.Tasks = append(next.Tasks, task)
			return nil
		})
		if err != nil {
			storeError(w, err)
			return
		}
		w.Header().Set("Location", "/tasks/"+strconv.Itoa(task.ID))
		answer(w, 201, task)
	})
	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := readID(w, r)
		if !ok {
			return
		}
		task, err := s.get(id)
		if err != nil {
			storeError(w, err)
			return
		}
		answer(w, 200, task)
	})
	mux.HandleFunc("PUT /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := readID(w, r)
		if !ok {
			return
		}
		fields, ok := readTask(w, r)
		if !ok {
			return
		}
		task := Task{ID: id, Title: fields.Title, Done: fields.Done}
		err := s.change(func(next *state) error {
			for i, t := range next.Tasks {
				if t.ID == id {
					next.Tasks[i] = task
					return nil
				}
			}
			return errNotFound
		})
		if err != nil {
			storeError(w, err)
			return
		}
		answer(w, 200, task)
	})
	mux.HandleFunc("DELETE /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, ok := readID(w, r)
		if !ok {
			return
		}
		err := s.change(func(next *state) error {
			for i, t := range next.Tasks {
				if t.ID == id {
					next.Tasks = append(next.Tasks[:i], next.Tasks[i+1:]...)
					return nil
				}
			}
			return errNotFound
		})
		if err != nil {
			storeError(w, err)
			return
		}
		w.WriteHeader(204)
	})
	return mux
}
func readID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		fail(w, 400, "invalid task ID")
		return 0, false
	}
	return id, true
}
func main() {
	path := os.Getenv("TASKS_FILE")
	if path == "" {
		path = "tasks.json"
	}
	s, err := openStore(path)
	if err != nil {
		log.Fatalf("open task store: %v", err)
	}
	address := "127.0.0.1:8080"
	srv := http.Server{Addr: address, Handler: routes(s), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	log.Printf("Task API listening on http://%s", address)
	log.Fatal(srv.ListenAndServe())
}
