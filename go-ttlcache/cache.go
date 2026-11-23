// Package ttlcache implements a fixed-capacity, expiring in-memory cache.
package ttlcache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type entry[V any] struct {
	key     string
	value   V
	expires time.Time
}

// Cache is safe for concurrent use. Construct it with New.
type Cache[V any] struct {
	mu       sync.Mutex
	capacity int
	now      func() time.Time
	entries  map[string]*list.Element
	order    list.List // most recently used at the front
}

// New constructs an empty cache. Pass nil to use the real clock. A custom
// clock is called under the cache lock and must not call back into the cache.
func New[V any](capacity int, now func() time.Time) (*Cache[V], error) {
	if capacity < 1 {
		return nil, errors.New("capacity must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return &Cache[V]{capacity: capacity, now: now, entries: make(map[string]*list.Element)}, nil
}

// Set inserts or replaces a value and starts its TTL from the current time.
func (c *Cache[V]) Set(key string, value V, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("TTL must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	next := entry[V]{key: key, value: value, expires: now.Add(ttl)}
	if e := c.entries[key]; e != nil {
		e.Value = next
		c.order.MoveToFront(e)
		return nil
	}
	if len(c.entries) == c.capacity {
		// Prefer to reclaim an expired entry over evicting a live entry.
		victim := c.order.Back()
		for e := victim; e != nil; e = e.Prev() {
			if !now.Before(e.Value.(entry[V]).expires) {
				victim = e
				break
			}
		}
		c.remove(victim)
	}
	c.entries[key] = c.order.PushFront(next)
	return nil
}

// Get returns a live value and marks it recently used. Getting a value does
// not extend its lifetime; an entry exactly at its deadline is expired.
func (c *Cache[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	e := c.entries[key]
	if e == nil {
		return zero, false
	}
	v := e.Value.(entry[V])
	if !c.now().Before(v.expires) {
		c.remove(e)
		return zero, false
	}
	c.order.MoveToFront(e)
	return v.value, true
}

// remove is called only while holding the cache lock.
func (c *Cache[V]) remove(e *list.Element) {
	delete(c.entries, e.Value.(entry[V]).key)
	c.order.Remove(e)
}
