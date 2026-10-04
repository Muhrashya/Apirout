package eventbus

import (
	"context"
	"io"
)

// EventHandler is the callback function invoked when an invalidation event is received
type EventHandler func(event InvalidationEvent)

// EventBus is the port interface for publishing and subscribing to invalidation events
type EventBus interface {
	io.Closer
	// Publish dispatches an invalidation event to all subscribers across the cluster
	Publish(ctx context.Context, event InvalidationEvent) error
	// Subscribe registers a handler function that processes incoming invalidation events
	Subscribe(ctx context.Context, handler EventHandler) error
}
