package gitlab

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      any
	expiresAt time.Time
}

// Cache is a simple TTL cache with a max size eviction policy.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	maxSize int
}

func NewCache(maxSize int) *Cache {
	return &Cache{entries: make(map[string]cacheEntry), maxSize: maxSize}
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.entries, key)
			c.mu.Unlock()
		}
		return nil, false
	}
	return e.data, true
}

func (c *Cache) Set(key string, data any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		// evict oldest
		var oldest string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldest == "" || v.expiresAt.Before(oldestTime) {
				oldest = k
				oldestTime = v.expiresAt
			}
		}
		if oldest != "" {
			delete(c.entries, oldest)
		}
	}
	c.entries[key] = cacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
}
