package gitlab

import (
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := NewCache(10)
	c.Set("key1", "value1", 5*time.Minute)

	v, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if v.(string) != "value1" {
		t.Errorf("expected 'value1', got %v", v)
	}
}

func TestCacheMiss(t *testing.T) {
	c := NewCache(10)
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := NewCache(10)
	c.Set("key1", "value1", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestCacheEviction(t *testing.T) {
	c := NewCache(3)
	c.Set("a", 1, 5*time.Minute)
	c.Set("b", 2, 5*time.Minute)
	c.Set("c", 3, 5*time.Minute)
	// This should evict one entry
	c.Set("d", 4, 5*time.Minute)

	// At least 3 entries should be accessible
	count := 0
	for _, k := range []string{"a", "b", "c", "d"} {
		if _, ok := c.Get(k); ok {
			count++
		}
	}
	if count > 3 {
		t.Errorf("expected max 3 entries after eviction, got %d", count)
	}
}
