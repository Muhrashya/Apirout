package cache

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)

// PooledConnection represents a cached target database connection
type PooledConnection struct {
	DB           *sql.DB
	LastUsed     time.Time
	ConnectionID string
}

// ConnectionPool manages reusable target database connections
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*PooledConnection
	maxIdle     time.Duration
	maxConns    int
	stopCh      chan struct{}
}

// NewConnectionPool creates a new connection pool manager
func NewConnectionPool(maxIdleTime time.Duration, maxConns int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		maxIdle:     maxIdleTime,
		maxConns:    maxConns,
		stopCh:      make(chan struct{}),
	}

	go pool.cleanupRoutine()
	return pool
}

// Get retrieves or creates a connection for the given target database profile
func (p *ConnectionPool) Get(connectionID, dbType, dsn string) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[connectionID]; exists {
		if err := conn.DB.Ping(); err == nil {
			conn.LastUsed = time.Now()
			return conn.DB, nil
		}
		conn.DB.Close()
		delete(p.connections, connectionID)
	}

	var driverName string
	switch dbType {
	case "postgres", "postgresql":
		driverName = "postgres"
	case "mysql":
		driverName = "mysql"
	case "mssql", "sqlserver":
		driverName = "sqlserver"
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", dbType)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(p.maxConns)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	p.connections[connectionID] = &PooledConnection{
		DB:           db,
		LastUsed:     time.Now(),
		ConnectionID: connectionID,
	}

	return db, nil
}

// Invalidate removes and closes a cached connection
func (p *ConnectionPool) Invalidate(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[connectionID]; exists {
		conn.DB.Close()
		delete(p.connections, connectionID)
	}
}

// Close closes all pooled connections
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.stopCh:
		// already stopped
	default:
		close(p.stopCh)
	}

	for id, conn := range p.connections {
		conn.DB.Close()
		delete(p.connections, id)
	}
}

func (p *ConnectionPool) cleanupRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.cleanup()
		}
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
