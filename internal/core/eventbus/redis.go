package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

// DefaultInvalidationTopic is the standard Redis Pub/Sub channel for cache invalidation events
const DefaultInvalidationTopic = "apirout:events:invalidation"

// RedisEventBus implements EventBus using Redis Pub/Sub for distributed clustering
type RedisEventBus struct {
	client redis.UniversalClient
	topic  string
	mu     sync.Mutex
	pubsub *redis.PubSub
	stopCh chan struct{}
	closed bool
}

// NewRedisEventBus creates a new Redis event bus instance
func NewRedisEventBus(client redis.UniversalClient, topic ...string) *RedisEventBus {
	t := DefaultInvalidationTopic
	if len(topic) > 0 && topic[0] != "" {
		t = topic[0]
	}
	return &RedisEventBus{
		client: client,
		topic:  t,
		stopCh: make(chan struct{}),
	}
}

// Publish serializes the event to JSON and publishes to the Redis topic
func (b *RedisEventBus) Publish(ctx context.Context, event InvalidationEvent) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return fmt.Errorf("redis event bus is closed")
	}
	b.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal invalidation event: %w", err)
	}

	return b.client.Publish(ctx, b.topic, data).Err()
}

// Subscribe listens to the Redis topic and invokes the handler for each event
func (b *RedisEventBus) Subscribe(ctx context.Context, handler EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("redis event bus is closed")
	}

	if b.pubsub == nil {
		b.pubsub = b.client.Subscribe(ctx, b.topic)
	}

	ch := b.pubsub.Channel()
	go func() {
		defer func() {
			b.mu.Lock()
			if b.pubsub != nil {
				_ = b.pubsub.Close()
				b.pubsub = nil
			}
			b.mu.Unlock()
		}()

		for {
			select {
			case <-b.stopCh:
				return
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var event InvalidationEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err == nil {
					handler(event)
				}
			}
		}
	}()

	return nil
}

// Close closes the Redis Pub/Sub subscription and stops background listener goroutines
func (b *RedisEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	select {
	case <-b.stopCh:
	default:
		close(b.stopCh)
	}
	if b.pubsub != nil {
		err := b.pubsub.Close()
		b.pubsub = nil
		return err
	}
	return nil
}
