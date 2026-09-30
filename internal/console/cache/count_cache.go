package cache

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// CountCacheEntry represents a cached count result
type CountCacheEntry struct {
	Total     int64
	CachedAt  time.Time
	ExpiresAt time.Time
}

// CountCache provides in-memory caching for count results
type CountCache struct {
	mu   sync.RWMutex
	data map[string]CountCacheEntry
	ttl  time.Duration
}

// NewCountCache creates a new count cache
func NewCountCache(ttl time.Duration) *CountCache {
	return &CountCache{
		data: make(map[string]CountCacheEntry),
		ttl:  ttl,
	}
}

// Get retrieves a count from cache
func (c *CountCache) Get(key string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return 0, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return 0, false
	}

	return entry.Total, true
}

// Set stores a count in cache
func (c *CountCache) Set(key string, total int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = CountCacheEntry{
		Total:     total,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// GenerateKey generates a cache key from endpoint ID and params
func (c *CountCache) GenerateKey(endpointID string, params map[string]interface{}) string {
	// Sort params to ensure consistent key
	keys := make([]string, 0, len(params))
	for k := range params {
		// Skip non-filtering params
		if k == "page" || k == "limit" || k == "__limit" || k == "__offset" || k == "token" || k == "access_token" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(endpointID)
	sb.WriteString(":")
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprintf("%v", params[k]))
		sb.WriteString(";")
	}
	return sb.String()
}

// Cleanup removes expired entries
func (c *CountCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.data {
		if now.After(v.ExpiresAt) {
			delete(c.data, k)
		}
	}
}

// StartCleanupRoutine starts background cleanup goroutine
func (c *CountCache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.Cleanup()
		}
	}()
}
