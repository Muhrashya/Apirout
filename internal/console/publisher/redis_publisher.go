package publisher

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/apirout/apirout/internal/core/eventbus"
	"github.com/apirout/apirout/internal/core/events"
)

// RedisPublisher emits configuration change events to Data Plane workers
type RedisPublisher struct {
	client *redis.Client
	bus    eventbus.EventBus
	logger *slog.Logger
}

// NewRedisPublisher creates a new publisher instance
func NewRedisPublisher(redisURL, redisHost, redisPort, redisPassword string, logger *slog.Logger) *RedisPublisher {
	var rdb *redis.Client
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			rdb = redis.NewClient(opt)
		}
	}

	if rdb == nil && redisHost != "" {
		addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: redisPassword,
			DB:       0,
		})
	}

	var bus eventbus.EventBus
	if rdb != nil {
		bus = eventbus.NewRedisEventBus(rdb)
	}

	return &RedisPublisher{
		client: rdb,
		bus:    bus,
		logger: logger,
	}
}

// NewPublisherWithBus creates a publisher backed by an explicit EventBus (ideal for tests)
func NewPublisherWithBus(bus eventbus.EventBus, logger *slog.Logger) *RedisPublisher {
	return &RedisPublisher{
		bus:    bus,
		logger: logger,
	}
}

// PublishEndpointChanged broadcasts an endpoint invalidation event
func (p *RedisPublisher) PublishEndpointChanged(action, endpointID, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Dispatch to typed EventBus if available
	if p.bus != nil {
		actionType := eventbus.ActionUpdated
		switch strings.ToUpper(action) {
		case "CREATE", "INSERT":
			actionType = eventbus.ActionCreated
		case "DELETE", "REMOVE":
			actionType = eventbus.ActionDeleted
		case "BULK_IMPORT", "PURGE":
			actionType = eventbus.ActionPurged
		}
		_ = p.bus.Publish(ctx, eventbus.NewInvalidationEvent(eventbus.EntityRoute, endpointID, actionType))
	}

	if p.client == nil {
		return nil
	}

	evt := events.NewEndpointInvalidateEvent(action, endpointID, path)
	payload, err := evt.Encode()
	if err != nil {
		return fmt.Errorf("encode endpoint event: %w", err)
	}

	if err := p.client.Publish(ctx, events.ChannelConfigEvents, payload).Err(); err != nil {
		if p.logger != nil {
			p.logger.Warn("Failed to publish endpoint change event to Redis", "error", err)
		}
		return err
	}

	if p.logger != nil {
		p.logger.Info("Broadcasted endpoint change event to Data Plane", "action", action, "path", path)
	}
	return nil
}

// PublishConnectionChanged broadcasts a target database connection reset event
func (p *RedisPublisher) PublishConnectionChanged(action, connectionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Dispatch to typed EventBus if available
	if p.bus != nil {
		actionType := eventbus.ActionUpdated
		if strings.ToUpper(action) == "DELETE" {
			actionType = eventbus.ActionDeleted
		}
		_ = p.bus.Publish(ctx, eventbus.NewInvalidationEvent(eventbus.EntityConnection, connectionID, actionType))
	}

	if p.client == nil {
		return nil
	}

	evt := events.NewConnectionInvalidateEvent(action, connectionID)
	payload, err := evt.Encode()
	if err != nil {
		return fmt.Errorf("encode connection event: %w", err)
	}

	if err := p.client.Publish(ctx, events.ChannelConnectionEvents, payload).Err(); err != nil {
		if p.logger != nil {
			p.logger.Warn("Failed to publish connection change event to Redis", "error", err)
		}
		return err
	}

	if p.logger != nil {
		p.logger.Info("Broadcasted connection reset event to Data Plane", "action", action, "connection_id", connectionID)
	}
	return nil
}

// Close closes the Redis connection and EventBus
func (p *RedisPublisher) Close() error {
	if p.bus != nil {
		_ = p.bus.Close()
	}
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
