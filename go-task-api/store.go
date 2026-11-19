package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var errNotFound = errors.New("task not found")

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
type state struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}
type store struct {
	mu    sync.Mutex
	path  string
	state state
}

func openStore(path string) (*store, error) {
	s := &store{path: path, state: state{NextID: 1, Tasks: []Task{}}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &s.state); err != nil {
		return nil, err
	}
	if s.state.NextID < 1 {
		return nil, errors.New("invalid next task ID")
	}
	seen := map[int]bool{}
	for _, task := range s.state.Tasks {
		if task.ID < 1 || task.ID >= s.state.NextID || seen[task.ID] {
			return nil, errors.New("invalid stored task ID")
		}
		seen[task.ID] = true
	}
	return s, nil
}

func (s *store) list() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Task{}, s.state.Tasks...)
}

func (s *store) get(id int) (Task, error) {
	for _, t := range s.list() {
		if t.ID == id {
			return t, nil
		}
	}
	return Task{}, errNotFound
}

// change stages a copy and commits it only after saving succeeds.
func (s *store) change(apply func(*state) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := state{NextID: s.state.NextID, Tasks: append([]Task{}, s.state.Tasks...)}
	if err := apply(&next); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".tasks-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := json.NewEncoder(f).Encode(next); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.state = next
	return nil
}
