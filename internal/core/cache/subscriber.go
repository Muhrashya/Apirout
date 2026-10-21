package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/apirout/apirout/internal/core/eventbus"
	"github.com/apirout/apirout/internal/core/events"
)

// RedisSubscriber manages real-time cache invalidation across distributed Gateway workers
type RedisSubscriber struct {
	client         *redis.Client
	bus            eventbus.EventBus
	endpointCache  *EndpointCache
	connectionPool *ManagedConnectionPool
	countCache            *CountCache
	logger                *slog.Logger
	stopCh                chan struct{}
	onRouteInvalidated    func()
	onTargetHealthChanged func(targetID, action string)
}

// SetOnRouteInvalidated registers a callback invoked when route invalidation events occur
func (s *RedisSubscriber) SetOnRouteInvalidated(fn func()) {
	s.onRouteInvalidated = fn
}

// SetOnTargetHealthChanged registers a callback invoked when target health state transitions occur
func (s *RedisSubscriber) SetOnTargetHealthChanged(fn func(targetID, action string)) {
	s.onTargetHealthChanged = fn
}

// Client returns the underlying Redis client if configured
func (s *RedisSubscriber) Client() *redis.Client {
	if s == nil {
		return nil
	}
	return s.client
}

// NewRedisSubscriber creates a new Redis subscriber instance
func NewRedisSubscriber(redisURL, redisHost, redisPort, redisPassword string,
	epCache *EndpointCache, connPool *ManagedConnectionPool, countCache *CountCache,
	logger *slog.Logger) *RedisSubscriber {

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

	return &RedisSubscriber{
		client:         rdb,
		bus:            bus,
		endpointCache:  epCache,
		connectionPool: connPool,
		countCache:     countCache,
		logger:         logger,
		stopCh:         make(chan struct{}),
	}
}

// NewSubscriberWithBus creates a subscriber backed by an explicit EventBus (ideal for testing)
func NewSubscriberWithBus(bus eventbus.EventBus, epCache *EndpointCache, connPool *ManagedConnectionPool, countCache *CountCache, logger *slog.Logger) *RedisSubscriber {
	return &RedisSubscriber{
		bus:            bus,
		endpointCache:  epCache,
		connectionPool: connPool,
		countCache:     countCache,
		logger:         logger,
		stopCh:         make(chan struct{}),
	}
}

// Start begins listening to EventBus and Redis Pub/Sub channels in background goroutine
func (s *RedisSubscriber) Start(ctx context.Context) {
	// Subscribe to typed EventBus if available
	if s.bus != nil {
		_ = s.bus.Subscribe(ctx, func(evt eventbus.InvalidationEvent) {
			s.HandleInvalidationEvent(evt)
		})
	}

	if s.client == nil {
		return
	}

	// Test ping first
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.client.Ping(pingCtx).Err(); err != nil {
		if s.logger != nil {
			s.logger.Warn("Redis server unreachable, running in standalone local-cache mode", "error", err)
		}
		return
	}

	if s.logger != nil {
		s.logger.Info("Connected to Redis Pub/Sub for distributed cache sync",
			"channels", []string{events.ChannelConfigEvents, events.ChannelConnectionEvents, eventbus.DefaultInvalidationTopic})
	}

	go s.listenLoop(ctx)
}

func (s *RedisSubscriber) listenLoop(ctx context.Context) {
	pubsub := s.client.Subscribe(ctx, events.ChannelConfigEvents, events.ChannelConnectionEvents)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			s.handleMessage(msg)
		}
	}
}

func (s *RedisSubscriber) handleMessage(msg *redis.Message) {
	switch msg.Channel {
	case events.ChannelConfigEvents:
		evt, err := events.DecodeEndpointInvalidateEvent([]byte(msg.Payload))
		if err != nil {
			if s.logger != nil {
				s.logger.Error("Failed to decode endpoint invalidate event", "payload", msg.Payload, "error", err)
			}
			return
		}

		if s.logger != nil {
			s.logger.Info("Received cache invalidation event from Control Plane",
				"action", evt.Action, "path", evt.Path, "endpoint_id", evt.EndpointID)
		}

		if evt.Path != "" {
			s.endpointCache.Invalidate(evt.Path)
		}
		if evt.EndpointID != "" {
			s.endpointCache.InvalidateByEndpointID(evt.EndpointID)
			s.countCache.Invalidate(evt.EndpointID)
		}
		if s.onRouteInvalidated != nil {
			s.onRouteInvalidated()
		}

	case events.ChannelConnectionEvents:
		evt, err := events.DecodeConnectionInvalidateEvent([]byte(msg.Payload))
		if err != nil {
			if s.logger != nil {
				s.logger.Error("Failed to decode connection invalidate event", "payload", msg.Payload, "error", err)
			}
			return
		}

		if s.logger != nil {
			s.logger.Info("Received target DB connection reset event", "connection_id", evt.ConnectionID)
		}
		s.connectionPool.Invalidate(evt.ConnectionID)
	}
}

// HandleInvalidationEvent processes strongly-typed invalidation events from EventBus
func (s *RedisSubscriber) HandleInvalidationEvent(evt eventbus.InvalidationEvent) {
	if s.logger != nil {
		s.logger.Info("Processing EventBus invalidation", "entity", evt.EntityType, "id", evt.EntityID, "action", evt.Action)
	}

	switch evt.EntityType {
	case eventbus.EntityRoute:
		if evt.EntityID != "" && evt.EntityID != "*" {
			s.endpointCache.InvalidateByEndpointID(evt.EntityID)
			s.countCache.Invalidate(evt.EntityID)
		} else {
			s.endpointCache.Clear()
			s.countCache.Clear()
		}
		if s.onRouteInvalidated != nil {
			s.onRouteInvalidated()
		}
	case eventbus.EntityConnection:
		if evt.EntityID != "" && evt.EntityID != "*" {
			s.connectionPool.Invalidate(evt.EntityID)
		} else {
			s.connectionPool.InvalidateAll()
		}
	case eventbus.EntityTarget:
		if s.onTargetHealthChanged != nil {
			s.onTargetHealthChanged(evt.EntityID, string(evt.Action))
		}
	case eventbus.EntityAll:
		s.endpointCache.Clear()
		s.countCache.Clear()
		s.connectionPool.InvalidateAll()
		if s.onRouteInvalidated != nil {
			s.onRouteInvalidated()
		}
	default:
		// For services, upstreams, plugins, flush cached endpoints
		s.endpointCache.Clear()
		s.countCache.Clear()
		if s.onRouteInvalidated != nil {
			s.onRouteInvalidated()
		}
	}
}

// Stop terminates the subscriber
func (s *RedisSubscriber) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	if s.bus != nil {
		_ = s.bus.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
}
