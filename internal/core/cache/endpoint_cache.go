package cache

import (
	"sync"
	"time"
)

// EndpointConfig contains cached metadata required for executing an endpoint
type EndpointConfig struct {
	EndpointID          string
	Method              string
	Path                string
	QueryID             *string
	QuerySQL            string
	QueryOrderBy        string
	ParamsSchema        string
	ConnectionID        *string
	Enabled             bool
	IsPublic            bool
	GroupID             *string
	HasLimit            bool
	DefaultLimit        int
	MaxLimit            int
	ReturnTotalCount    bool
	AllowDynamicFilters bool
	CountSQLText        string
	QueryDefaultLimit   int

	CachedAt time.Time
}

// EndpointCache stores endpoint configs in memory with TTL
type EndpointCache struct {
	mu     sync.RWMutex
	items  map[string]*EndpointConfig
	ttl    time.Duration
	stopCh chan struct{}
}

// NewEndpointCache creates a new in-memory endpoint cache
func NewEndpointCache(ttl time.Duration) *EndpointCache {
	return &EndpointCache{
		items:  make(map[string]*EndpointConfig),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
}

// Get retrieves an endpoint config by URL path
func (c *EndpointCache) Get(path string) (*EndpointConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[path]
	if !exists {
		return nil, false
	}

	if time.Since(item.CachedAt) > c.ttl {
		return nil, false
	}

	return item, true
}

// Set stores an endpoint config
func (c *EndpointCache) Set(path string, config *EndpointConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	config.CachedAt = time.Now()
	c.items[path] = config
}

// Invalidate removes a specific path from cache
func (c *EndpointCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, path)
}

// InvalidateByEndpointID removes any cached entry with matching endpoint ID
func (c *EndpointCache) InvalidateByEndpointID(endpointID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, item := range c.items {
		if item.EndpointID == endpointID {
			delete(c.items, path)
		}
	}
}

// InvalidateByID is an alias for InvalidateByEndpointID
func (c *EndpointCache) InvalidateByID(endpointID string) {
	c.InvalidateByEndpointID(endpointID)
}

// InvalidateByQuery invalidates cache entries using a specific query ID
func (c *EndpointCache) InvalidateByQuery(queryID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, config := range c.items {
		if config.QueryID != nil && *config.QueryID == queryID {
			delete(c.items, path)
		}
	}
}

// InvalidateByConnection invalidates cache entries using a specific connection ID
func (c *EndpointCache) InvalidateByConnection(connectionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, config := range c.items {
		if config.ConnectionID != nil && *config.ConnectionID == connectionID {
			delete(c.items, path)
		}
	}
}

// Clear flushes all cached configs
func (c *EndpointCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*EndpointConfig)
}

// InvalidateAll clears all cached endpoints
func (c *EndpointCache) InvalidateAll() {
	c.Clear()
}

// Size returns number of cached entries
func (c *EndpointCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// StartCleanupRoutine periodically removes expired entries
func (c *EndpointCache) StartCleanupRoutine(interval time.Duration) {
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

func (c *EndpointCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for path, item := range c.items {
		if now.Sub(item.CachedAt) > c.ttl {
			delete(c.items, path)
		}
	}
}

// Stop stops the background cleanup routine
func (c *EndpointCache) Stop() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}
