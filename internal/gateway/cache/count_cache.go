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
	mu     sync.RWMutex
	data   map[string]CountCacheEntry
	ttl    time.Duration
	stopCh chan struct{}
}

// NewCountCache creates a new count cache
func NewCountCache(ttl time.Duration) *CountCache {
	return &CountCache{
		data:   make(map[string]CountCacheEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
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

// Invalidate removes cached counts for an endpoint
func (c *CountCache) Invalidate(endpointID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := endpointID + ":"
	for key := range c.data {
		if key == endpointID || strings.HasPrefix(key, prefix) {
			delete(c.data, key)
		}
	}
}

// Clear flushes all cached counts
func (c *CountCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]CountCacheEntry)
}

// GenerateKey creates a deterministic cache key based on params
func (c *CountCache) GenerateKey(endpointID string, params map[string]interface{}) string {
	if len(params) == 0 {
		return endpointID + ":default"
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "limit" || k == "offset" || k == "page" || strings.HasPrefix(k, "__") {
			continue
		}
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(endpointID)
	for _, k := range keys {
		sb.WriteString(":")
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprintf("%v", params[k]))
	}

	return sb.String()
}

// StartCleanupRoutine runs periodic cleanup of expired counts
func (c *CountCache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopCh:
				return
			case <-ticker.C:
				c.cleanup()
			}
		}
	}()
}

func (c *CountCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
		}
	}
}

// Stop stops the cleanup routine
func (c *CountCache) Stop() {
	close(c.stopCh)
}
