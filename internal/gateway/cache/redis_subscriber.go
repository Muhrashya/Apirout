package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/apirout/apirout/internal/core/events"
)

// RedisSubscriber manages real-time cache invalidation across distributed Gateway workers
type RedisSubscriber struct {
	client         *redis.Client
	endpointCache  *EndpointCache
	connectionPool *ConnectionPool
	countCache     *CountCache
	logger         *slog.Logger
	stopCh         chan struct{}
}

// NewRedisSubscriber creates a new Redis subscriber instance
func NewRedisSubscriber(redisURL, redisHost, redisPort, redisPassword string,
	epCache *EndpointCache, connPool *ConnectionPool, countCache *CountCache,
	logger *slog.Logger) *RedisSubscriber {

	var rdb *redis.Client
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			rdb = redis.NewClient(opt)
		}
	}

	if rdb == nil {
		addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: redisPassword,
			DB:       0,
		})
	}

	return &RedisSubscriber{
		client:         rdb,
		endpointCache:  epCache,
		connectionPool: connPool,
		countCache:     countCache,
		logger:         logger,
		stopCh:         make(chan struct{}),
	}
}

// Start begins listening to Redis Pub/Sub channels in background goroutine
func (s *RedisSubscriber) Start(ctx context.Context) {
	// Test ping first
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := s.client.Ping(pingCtx).Err(); err != nil {
		s.logger.Warn("Redis server unreachable, running in standalone local-cache mode", "error", err)
		return
	}

	s.logger.Info("Connected to Redis Pub/Sub for distributed cache sync",
		"channels", []string{events.ChannelConfigEvents, events.ChannelConnectionEvents})

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
			s.logger.Error("Failed to decode endpoint invalidate event", "payload", msg.Payload, "error", err)
			return
		}

		s.logger.Info("Received cache invalidation event from Control Plane",
			"action", evt.Action, "path", evt.Path, "endpoint_id", evt.EndpointID)

		if evt.Path != "" {
			s.endpointCache.Invalidate(evt.Path)
		}
		if evt.EndpointID != "" {
			s.endpointCache.InvalidateByEndpointID(evt.EndpointID)
			s.countCache.Invalidate(evt.EndpointID)
		}

	case events.ChannelConnectionEvents:
		evt, err := events.DecodeConnectionInvalidateEvent([]byte(msg.Payload))
		if err != nil {
			s.logger.Error("Failed to decode connection invalidate event", "payload", msg.Payload, "error", err)
			return
		}

		s.logger.Info("Received target DB connection reset event", "connection_id", evt.ConnectionID)
		s.connectionPool.Invalidate(evt.ConnectionID)
	}
}

// Stop terminates the subscriber
func (s *RedisSubscriber) Stop() {
	close(s.stopCh)
	if s.client != nil {
		_ = s.client.Close()
	}
}
