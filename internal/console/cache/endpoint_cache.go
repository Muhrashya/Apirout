package cache

import (
	"sync"
	"time"
)

// EndpointConfig represents cached endpoint configuration
type EndpointConfig struct {
	EndpointID        string
	Method            string
	Path              string
	QueryID           *string
	QuerySQL          string
	QueryOrderBy      string
	ParamsSchema      string
	ConnectionID      *string
	Enabled           bool
	IsPublic          bool
	GroupID           *string
	HasLimit          bool
	DefaultLimit      int
	MaxLimit          int
	ReturnTotalCount    bool
	AllowDynamicFilters bool
	CountSQLText        string
	QueryDefaultLimit   int
	CachedAt          time.Time
}

// EndpointCache provides in-memory caching for endpoint configurations
type EndpointCache struct {
	mu      sync.RWMutex
	data    map[string]*EndpointConfig // key: endpoint path
	ttl     time.Duration
	enabled bool
}

// NewEndpointCache creates a new endpoint cache
func NewEndpointCache(ttl time.Duration) *EndpointCache {
	return &EndpointCache{
		data:    make(map[string]*EndpointConfig),
		ttl:     ttl,
		enabled: true,
	}
}

// Get retrieves endpoint config from cache
func (c *EndpointCache) Get(path string) (*EndpointConfig, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	config, exists := c.data[path]
	if !exists {
		return nil, false
	}

	// Check TTL
	if time.Since(config.CachedAt) > c.ttl {
		return nil, false
	}

	return config, true
}

// Set stores endpoint config in cache
func (c *EndpointCache) Set(path string, config *EndpointConfig) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	config.CachedAt = time.Now()
	c.data[path] = config
}

// Invalidate removes a specific endpoint from cache
func (c *EndpointCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, path)
}

// InvalidateByID removes endpoint by ID (searches all entries)
func (c *EndpointCache) InvalidateByID(endpointID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, config := range c.data {
		if config.EndpointID == endpointID {
			delete(c.data, path)
			return
		}
	}
}

// InvalidateAll clears all cached endpoints
func (c *EndpointCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*EndpointConfig)
}

// InvalidateByQuery invalidates cache entries using a specific query
func (c *EndpointCache) InvalidateByQuery(queryID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, config := range c.data {
		if config.QueryID != nil && *config.QueryID == queryID {
			delete(c.data, path)
		}
	}
}

// InvalidateByConnection invalidates cache entries using a specific connection
func (c *EndpointCache) InvalidateByConnection(connectionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path, config := range c.data {
		if config.ConnectionID != nil && *config.ConnectionID == connectionID {
			delete(c.data, path)
		}
	}
}

// Size returns number of cached entries
func (c *EndpointCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Cleanup removes expired entries
func (c *EndpointCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for path, config := range c.data {
		if now.Sub(config.CachedAt) > c.ttl {
			delete(c.data, path)
		}
	}
}

// StartCleanupRoutine starts background cleanup goroutine
func (c *EndpointCache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.Cleanup()
		}
	}()
}
