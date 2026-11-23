package ttlcache

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestExpiryAndReplacement(t *testing.T) {
	now := time.Unix(0, 0)
	c, err := New[string](2, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	set := func(value string, ttl time.Duration) {
		t.Helper()
		if err := c.Set("k", value, ttl); err != nil {
			t.Fatal(err)
		}
	}
	set("old", time.Second)
	now = now.Add(time.Second)
	if _, ok := c.Get("k"); ok {
		t.Fatal("hit at expiration")
	}
	set("new", time.Second)
	set("replacement", 2*time.Second)
	now = now.Add(time.Second)
	if got, ok := c.Get("k"); !ok || got != "replacement" {
		t.Fatalf("%q, %v", got, ok)
	}
	now = now.Add(time.Second)
	if _, ok := c.Get("k"); ok {
		t.Fatal("Get must not extend TTL")
	}
}

func TestCapacityReclaimsExpiredThenEvictsLRU(t *testing.T) {
	now := time.Unix(0, 0)
	c, _ := New[int](2, func() time.Time { return now })
	set := func(k string, ttl time.Duration) {
		t.Helper()
		if err := c.Set(k, 1, ttl); err != nil {
			t.Fatal(err)
		}
	}
	set("live", time.Minute)
	set("expired", time.Second)
	now = now.Add(time.Second)
	set("new", time.Minute)
	if _, ok := c.Get("live"); !ok {
		t.Fatal("live entry should survive an expired entry")
	}
	set("added", time.Minute)
	if _, ok := c.Get("new"); ok {
		t.Fatal("least recently used entry should be evicted")
	}
	if _, ok := c.Get("live"); !ok {
		t.Fatal("recently used entry was evicted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c, _ := New[int](32, nil)
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			k := strconv.Itoa(i)
			if err := c.Set(k, i, time.Hour); err != nil {
				t.Error(err)
				return
			}
			if got, ok := c.Get(k); !ok || got != i {
				t.Errorf("%s: got %d, %v", k, got, ok)
			}
		}()
	}
	group.Wait()
	if len(c.entries) > 32 {
		t.Fatal("capacity exceeded")
	}
}

func TestInvalidArguments(t *testing.T) {
	if _, err := New[int](0, nil); err == nil {
		t.Error("accepted zero capacity")
	}
	c, _ := New[int](1, nil)
	if err := c.Set("x", 1, 0); err == nil {
		t.Error("accepted zero TTL")
	}
}
