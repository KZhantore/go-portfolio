package main

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func await(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worker")
	}
}

func TestBoundedQueueAndDrain(t *testing.T) {
	q, err := NewQueue(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	entered, release, second := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	defer q.Shutdown()
	defer releaseOnce()
	if err = q.TrySubmit(func() { close(entered); <-release }); err != nil {
		t.Fatal(err)
	}
	await(t, entered)
	if err = q.TrySubmit(func() { close(second) }); err != nil {
		t.Fatal(err)
	}
	if err = q.TrySubmit(func() {}); !errors.Is(err, ErrFull) {
		t.Fatalf("want full, got %v", err)
	}
	stopped := make(chan struct{})
	go func() { q.Shutdown(); close(stopped) }()
	select {
	case <-stopped:
		t.Fatal("shutdown returned before releasing job")
	default:
	}
	releaseOnce()
	await(t, stopped)
	select {
	case <-second:
	default:
		t.Fatal("accepted job was lost")
	}
	if err = q.TrySubmit(func() {}); !errors.Is(err, ErrClosed) {
		t.Fatalf("want closed, got %v", err)
	}
}

func TestConcurrentWorkers(t *testing.T) {
	q, err := NewQueue(2, 6)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}, 6), make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	defer q.Shutdown()
	defer releaseOnce()
	var active, peak, done atomic.Int32
	for i := 0; i < 6; i++ {
		if err = q.TrySubmit(func() {
			n := active.Add(1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			entered <- struct{}{}
			<-release
			active.Add(-1)
			done.Add(1)
		}); err != nil {
			t.Fatal(err)
		}
	}
	await(t, entered)
	await(t, entered)
	releaseOnce()
	var closers sync.WaitGroup
	for i := 0; i < 2; i++ {
		closers.Add(1)
		go func() { defer closers.Done(); q.Shutdown() }()
	}
	closers.Wait()
	if peak.Load() != 2 || done.Load() != 6 {
		t.Fatalf("peak %d, completed %d", peak.Load(), done.Load())
	}
}

func TestValidationAndDemo(t *testing.T) {
	if _, err := NewQueue(0, 1); err == nil {
		t.Error("accepted zero workers")
	}
	if _, err := NewQueue(1, 0); err == nil {
		t.Error("accepted zero capacity")
	}
	q, _ := NewQueue(1, 1)
	defer q.Shutdown()
	if q.TrySubmit(nil) == nil {
		t.Error("accepted nil job")
	}
	var out, diagnostics bytes.Buffer
	if code := run([]string{"-workers=2", "-jobs=3"}, &out, &diagnostics); code != 0 || out.String() != "Job 1: 1\nJob 2: 4\nJob 3: 9\n" {
		t.Fatalf("exit=%d out=%q diagnostics=%q", code, out.String(), diagnostics.String())
	}
	if code := run([]string{"-workers=0"}, &out, &diagnostics); code != 2 {
		t.Errorf("bad configuration exit=%d", code)
	}
}
