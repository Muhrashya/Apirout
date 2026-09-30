package cache

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// PooledConnection represents a cached database connection
type PooledConnection struct {
	DB           *sql.DB
	LastUsed     time.Time
	ConnectionID string
}

// ConnectionPool manages reusable database connections per connection profile
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*PooledConnection
	maxIdle     time.Duration
	maxConns    int
}

// NewConnectionPool creates a new connection pool manager
func NewConnectionPool(maxIdleTime time.Duration, maxConns int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		maxIdle:     maxIdleTime,
		maxConns:    maxConns,
	}

	// Start background cleanup
	go pool.cleanupRoutine()

	return pool
}

// Get retrieves or creates a connection for the given connection profile
func (p *ConnectionPool) Get(connectionID, dbType, dsn string) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check existing connection
	if conn, exists := p.connections[connectionID]; exists {
		// Verify connection is still alive
		if err := conn.DB.Ping(); err == nil {
			conn.LastUsed = time.Now()
			return conn.DB, nil
		}
		// Connection dead, remove it
		conn.DB.Close()
		delete(p.connections, connectionID)
	}

	// Create new connection
	var driverName string
	switch dbType {
	case "postgres":
		driverName = "postgres"
	case "mysql":
		driverName = "mysql"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	// Configure pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// Store in pool
	p.connections[connectionID] = &PooledConnection{
		DB:           db,
		LastUsed:     time.Now(),
		ConnectionID: connectionID,
	}

	return db, nil
}

// Invalidate removes and closes a specific connection
func (p *ConnectionPool) Invalidate(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[connectionID]; exists {
		conn.DB.Close()
		delete(p.connections, connectionID)
	}
}

// InvalidateAll closes all connections
func (p *ConnectionPool) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, conn := range p.connections {
		conn.DB.Close()
		delete(p.connections, id)
	}
}

// Size returns number of pooled connections
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// cleanupRoutine removes idle connections periodically
func (p *ConnectionPool) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

func (p *ConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, conn := range p.connections {
		if now.Sub(conn.LastUsed) > p.maxIdle {
			conn.DB.Close()
			delete(p.connections, id)
		}
	}
}

// Close closes all connections and stops the pool
func (p *ConnectionPool) Close() {
	p.InvalidateAll()
}
