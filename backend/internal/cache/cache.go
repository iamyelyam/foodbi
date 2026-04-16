// Package cache provides a simple in-memory TTL cache for read-heavy endpoints.
//
// Invariants:
//   - Safe for concurrent use.
//   - Entries auto-expire after their TTL — no stale reads.
//   - Bounded by periodic GC (every minute).
//
// Use for dashboard summary, stats, and similar aggregate queries that are
// expensive to compute but tolerate 1-5 min staleness. Do NOT cache user-specific
// data (orders list, stock) — those require real-time consistency.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     interface{}
	expiresAt time.Time
}

// Cache is a concurrent TTL-expiring map.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]entry
	stopGC  chan struct{}
}

// New creates a cache and starts its GC goroutine.
// Call Close() on shutdown to stop the GC goroutine.
func New() *Cache {
	c := &Cache{
		items:  make(map[string]entry),
		stopGC: make(chan struct{}),
	}
	go c.gcLoop()
	return c
}

// Get returns (value, true) if key exists and not expired, else (nil, false).
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		// Expired — delete lazily
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.value, true
}

// Set stores value under key with the given TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	c.items[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Delete removes the key from the cache (used for invalidation after writes).
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// InvalidatePrefix deletes all entries whose key starts with the given prefix.
// Used to invalidate all cache entries for a company after sync.
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Close stops the GC goroutine.
func (c *Cache) Close() {
	close(c.stopGC)
}

func (c *Cache) gcLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopGC:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, e := range c.items {
				if now.After(e.expiresAt) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
