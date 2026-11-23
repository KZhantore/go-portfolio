package main

import (
	"fmt"
	"log"
	"time"

	"github.com/KZhantore/go-portfolio/go-ttlcache"
)

func main() {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c, err := ttlcache.New[string](2, func() time.Time { return now })
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Set("language", "Go", 2*time.Second); err != nil {
		log.Fatal(err)
	}
	value, hit := c.Get("language")
	fmt.Printf("Before expiry: %s (hit: %t)\n", value, hit)
	now = now.Add(2 * time.Second)
	_, hit = c.Get("language")
	fmt.Printf("At expiry: hit: %t\n", hit)
}
