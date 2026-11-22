package main

import (
	"errors"
	"sync"
)

var (
	ErrFull   = errors.New("queue is full")
	ErrClosed = errors.New("queue is closed")
)

type Queue struct {
	mu      sync.Mutex
	closed  bool
	jobs    chan func()
	workers sync.WaitGroup
}

func NewQueue(workers, capacity int) (*Queue, error) {
	if workers < 1 || capacity < 1 {
		return nil, errors.New("workers and capacity must be positive")
	}
	q := &Queue{jobs: make(chan func(), capacity)}
	q.workers.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer q.workers.Done()
			for job := range q.jobs {
				job()
			}
		}()
	}
	return q, nil
}

// TrySubmit queues a job immediately or returns a clear rejection.
func (q *Queue) TrySubmit(job func()) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	if job == nil {
		return errors.New("nil job")
	}
	select {
	case q.jobs <- job:
		return nil
	default:
		return ErrFull
	}
}

// Shutdown stops admissions and waits for every accepted job to finish.
// Call from outside a worker; concurrent calls are safe.
func (q *Queue) Shutdown() {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		close(q.jobs)
	}
	q.mu.Unlock()
	q.workers.Wait()
}
