package cache

import (
	"sync/atomic"
	"testing"
	"time"
)

type mockTunnel struct {
	stopped int32
}

func (m *mockTunnel) Stop() error {
	atomic.AddInt32(&m.stopped, 1)
	return nil
}

func TestManagedConnectionPool_LifecycleAndTeardown(t *testing.T) {
	pool := NewManagedConnectionPool(100*time.Millisecond, 5)
	defer pool.Close()

	tunnel := &mockTunnel{}
	connID := "conn-1"

	pool.mu.Lock()
	pool.connections[connID] = &PooledConnection{
		DB:           nil, // nil DB for unit test
		Tunnel:       tunnel,
		LastUsed:     time.Now(),
		ConnectionID: connID,
	}
	pool.mu.Unlock()

	// Invalidate should call Tunnel.Stop()
	pool.Invalidate(connID)

	if atomic.LoadInt32(&tunnel.stopped) != 1 {
		t.Fatalf("expected tunnel to be stopped once on invalidate, got %d", tunnel.stopped)
	}

	pool.mu.RLock()
	_, exists := pool.connections[connID]
	pool.mu.RUnlock()
	if exists {
		t.Fatalf("expected connection %s to be deleted from pool", connID)
	}
}

func TestManagedConnectionPool_CloseTeardown(t *testing.T) {
	pool := NewManagedConnectionPool(1*time.Minute, 5)

	tunnel1 := &mockTunnel{}
	tunnel2 := &mockTunnel{}

	pool.mu.Lock()
	pool.connections["conn-1"] = &PooledConnection{
		Tunnel:       tunnel1,
		LastUsed:     time.Now(),
		ConnectionID: "conn-1",
	}
	pool.connections["conn-2"] = &PooledConnection{
		Tunnel:       tunnel2,
		LastUsed:     time.Now(),
		ConnectionID: "conn-2",
	}
	pool.mu.Unlock()

	pool.Close()

	if atomic.LoadInt32(&tunnel1.stopped) != 1 {
		t.Errorf("expected tunnel1 to be stopped on Close, got %d", tunnel1.stopped)
	}
	if atomic.LoadInt32(&tunnel2.stopped) != 1 {
		t.Errorf("expected tunnel2 to be stopped on Close, got %d", tunnel2.stopped)
	}
}

func TestManagedConnectionPool_CleanupExpired(t *testing.T) {
	pool := NewManagedConnectionPool(50*time.Millisecond, 5)
	defer pool.Close()

	tunnel := &mockTunnel{}
	connID := "expired-conn"

	pool.mu.Lock()
	pool.connections[connID] = &PooledConnection{
		Tunnel:       tunnel,
		LastUsed:     time.Now().Add(-100 * time.Millisecond), // expired
		ConnectionID: connID,
	}
	pool.mu.Unlock()

	pool.cleanup()

	if atomic.LoadInt32(&tunnel.stopped) != 1 {
		t.Fatalf("expected expired tunnel to be stopped on cleanup, got %d", tunnel.stopped)
	}
}

func TestManagedConnectionPool_DriverValidation(t *testing.T) {
	pool := NewManagedConnectionPool(1*time.Minute, 5)
	defer pool.Close()

	_, err := pool.Get("invalid-conn", "unsupported-driver", "invalid-dsn")
	if err == nil {
		t.Fatal("expected error for unsupported driver, got nil")
	}
}

func TestCountCache(t *testing.T) {
	c := NewCountCache(500 * time.Millisecond)
	defer c.Stop()

	key := c.GenerateKey("ep-1", map[string]interface{}{"status": "active", "limit": 10})
	c.Set(key, 42)

	val, ok := c.Get(key)
	if !ok || val != 42 {
		t.Fatalf("expected count 42, got %d (ok=%v)", val, ok)
	}

	c.Invalidate("ep-1")
	_, ok = c.Get(key)
	if ok {
		t.Fatal("expected key to be invalidated")
	}
}
