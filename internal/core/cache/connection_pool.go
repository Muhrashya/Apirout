package cache

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/apirout/apirout/internal/core/tunnel"
)

// TunnelStopper defines an interface for managing background tunnel lifecycles
type TunnelStopper interface {
	Stop() error
}

// PooledConnection represents a cached target database connection
type PooledConnection struct {
	DB           *sql.DB
	Tunnel       TunnelStopper
	LastUsed     time.Time
	ConnectionID string
}

// ManagedConnectionPool manages reusable target database connections and their SSH bastion tunnels
type ManagedConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*PooledConnection
	tunnels     map[string]*tunnel.SSHTunnel
	maxIdle     time.Duration
	maxConns    int
	stopCh      chan struct{}
}

// NewManagedConnectionPool creates a new managed connection pool
func NewManagedConnectionPool(maxIdleTime time.Duration, maxConns int) *ManagedConnectionPool {
	pool := &ManagedConnectionPool{
		connections: make(map[string]*PooledConnection),
		tunnels:     make(map[string]*tunnel.SSHTunnel),
		maxIdle:     maxIdleTime,
		maxConns:    maxConns,
		stopCh:      make(chan struct{}),
	}

	go pool.cleanupRoutine()
	return pool
}

// Get retrieves or creates a connection for direct database connections
func (p *ManagedConnectionPool) Get(connectionID, dbType, dsn string) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[connectionID]; exists {
		if err := conn.DB.Ping(); err == nil {
			conn.LastUsed = time.Now()
			return conn.DB, nil
		}
		p.teardownConnection(conn)
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
	case "trino":
		driverName = "trino"
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

// GetTunnelPort returns a reusable local port forwarding through an SSH bastion, managing its lifecycle
func (p *ManagedConnectionPool) GetTunnelPort(connectionID string, cfg tunnel.Config) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Check if tunnel is already active
	if existing, exists := p.tunnels[connectionID]; exists && existing != nil {
		if existing.LocalPort > 0 {
			return existing.LocalPort, nil
		}
	}

	// 2. Instantiate and start dedicated tunnel
	sshTun, err := tunnel.NewSSHTunnel(cfg)
	if err != nil {
		return 0, fmt.Errorf("init ssh tunnel for connection %s: %w", connectionID, err)
	}

	localPort, err := sshTun.Start()
	if err != nil {
		return 0, fmt.Errorf("start ssh tunnel for connection %s: %w", connectionID, err)
	}

	p.tunnels[connectionID] = sshTun
	return localPort, nil
}

// Invalidate removes and cleanly closes a cached connection and its tunnel
func (p *ManagedConnectionPool) Invalidate(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[connectionID]; exists {
		p.teardownConnection(conn)
		delete(p.connections, connectionID)
	}

	if tun, exists := p.tunnels[connectionID]; exists {
		if tun != nil {
			_ = tun.Stop()
		}
		delete(p.tunnels, connectionID)
	}
}

// InvalidateAll closes and purges all pooled connections and tunnels
func (p *ManagedConnectionPool) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, conn := range p.connections {
		p.teardownConnection(conn)
		delete(p.connections, id)
	}

	for id, tun := range p.tunnels {
		if tun != nil {
			_ = tun.Stop()
		}
		delete(p.tunnels, id)
	}
}

// Close closes all pooled connections, stops all tunnels, and terminates background cleanup
func (p *ManagedConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case <-p.stopCh:
		// already stopped
	default:
		close(p.stopCh)
	}

	for id, conn := range p.connections {
		p.teardownConnection(conn)
		delete(p.connections, id)
	}

	for id, tun := range p.tunnels {
		if tun != nil {
			_ = tun.Stop()
		}
		delete(p.tunnels, id)
	}
}

func (p *ManagedConnectionPool) teardownConnection(conn *PooledConnection) {
	if conn.DB != nil {
		_ = conn.DB.Close()
	}
	if conn.Tunnel != nil {
		_ = conn.Tunnel.Stop()
	}
}

func (p *ManagedConnectionPool) cleanupRoutine() {
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

func (p *ManagedConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, conn := range p.connections {
		if now.Sub(conn.LastUsed) > p.maxIdle {
			p.teardownConnection(conn)
			delete(p.connections, id)

			// Clean up any associated tunnel for this idle connection
			if tun, exists := p.tunnels[id]; exists {
				if tun != nil {
					_ = tun.Stop()
				}
				delete(p.tunnels, id)
			}
		}
	}
}
