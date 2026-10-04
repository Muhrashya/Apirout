package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/apirout/apirout/internal/core/eventbus"
	"github.com/apirout/apirout/internal/gateway/cache"
)

func TestControlToDataPlaneSync_InMemoryIntegration(t *testing.T) {
	// 1. Setup shared in-memory EventBus
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	// 2. Setup Data Plane Caches
	epCache := cache.NewEndpointCache(5 * time.Minute)
	connPool := cache.NewConnectionPool(10*time.Minute, 10)
	countCache := cache.NewCountCache(5 * time.Minute)

	// Pre-populate Data Plane caches
	epCache.Set("/api/v1/users", &cache.EndpointConfig{
		EndpointID: "ep-users-123",
		Path:       "/api/v1/users",
	})
	countCache.Set("ep-users-123", 42)

	// Verify caches are warm
	if _, found := epCache.Get("/api/v1/users"); !found {
		t.Fatalf("expected endpoint to be cached")
	}
	if val, found := countCache.Get("ep-users-123"); !found || val != 42 {
		t.Fatalf("expected count 42 to be cached, got: %v", val)
	}

	// 3. Setup Data Plane Subscriber connected to EventBus
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber := cache.NewSubscriberWithBus(bus, epCache, connPool, countCache, nil)
	subscriber.Start(ctx)
	defer subscriber.Stop()

	// 4. Setup Control Plane Publisher connected to same EventBus
	pub := publisher.NewPublisherWithBus(bus, nil)
	defer pub.Close()

	// 5. Control Plane: Simulate route update in Console
	err := pub.PublishEndpointChanged("UPDATE", "ep-users-123", "/api/v1/users")
	if err != nil {
		t.Fatalf("failed to publish endpoint changed: %v", err)
	}

	// 6. Verify Data Plane cache was automatically invalidated in-memory
	if _, found := epCache.Get("/api/v1/users"); found {
		t.Errorf("expected endpoint cache to be invalidated after Control Plane update")
	}
	if _, found := countCache.Get("ep-users-123"); found {
		t.Errorf("expected count cache to be invalidated after Control Plane update")
	}

	// 7. Control Plane: Simulate target connection reset
	err = pub.PublishConnectionChanged("UPDATE", "conn-pg-primary")
	if err != nil {
		t.Fatalf("failed to publish connection changed: %v", err)
	}
}
