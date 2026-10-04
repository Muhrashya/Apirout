package eventbus

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryEventBus is an in-process, zero-dependency EventBus adapter ideal for testing and single-node mode
type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers []EventHandler
	closed   bool
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Publish delivers the event to all registered in-memory handlers
func (b *InMemoryEventBus) Publish(ctx context.Context, event InvalidationEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return fmt.Errorf("in-memory event bus is closed")
	}

	for _, h := range b.handlers {
		if h != nil {
			h(event)
		}
	}
	return nil
}

// Subscribe registers a new event handler
func (b *InMemoryEventBus) Subscribe(ctx context.Context, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("in-memory event bus is closed")
	}

	if handler != nil {
		b.handlers = append(b.handlers, handler)
	}
	return nil
}

// Close closes the in-memory event bus and clears handlers
func (b *InMemoryEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	b.handlers = nil
	return nil
}
