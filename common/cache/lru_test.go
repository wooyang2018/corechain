package cache

import (
	"testing"
)

func TestLRU(t *testing.T) {
	// Create lru cache
	cache := NewLRUCache(10000)
	// Add a key-value pair to cache
	cache.Add("key", "val")
	// Get value by key
	value, ok := cache.Get("key")
	if ok {
		t.Log(value)
	}
	// Delete value by key
	cache.Del("key")
	// Get count of items in cache
	count := cache.Len()
	t.Log(count)
}
