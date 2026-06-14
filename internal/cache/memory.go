package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     any
	expiresAt time.Time
}

// MemoryCache is a thread-safe in-memory TTL cache (L1).
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// NewMemoryCache creates an empty TTL cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]entry),
	}
}

// Get returns a cached value if present and not expired.
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.value, true
}

// Set stores value with the given TTL.
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	c.entries[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}

// Delete removes a key from the cache.
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}
