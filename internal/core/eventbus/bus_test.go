package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

func TestInMemoryEventBus(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	var received1, received2 InvalidationEvent

	err := bus.Subscribe(ctx, func(event InvalidationEvent) {
		received1 = event
		wg.Done()
	})
	if err != nil {
		t.Fatalf("failed to subscribe handler 1: %v", err)
	}

	err = bus.Subscribe(ctx, func(event InvalidationEvent) {
		received2 = event
		wg.Done()
	})
	if err != nil {
		t.Fatalf("failed to subscribe handler 2: %v", err)
	}

	testEvent := NewInvalidationEvent(EntityRoute, "route-123", ActionUpdated)

	err = bus.Publish(ctx, testEvent)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	wg.Wait()

	if received1.EntityID != "route-123" || received1.EntityType != EntityRoute || received1.Action != ActionUpdated {
		t.Errorf("handler 1 received unexpected event: %+v", received1)
	}
	if received2.EntityID != "route-123" || received2.EntityType != EntityRoute || received2.Action != ActionUpdated {
		t.Errorf("handler 2 received unexpected event: %+v", received2)
	}
}

func TestInMemoryEventBus_Closed(t *testing.T) {
	bus := NewInMemoryEventBus()
	bus.Close()

	ctx := context.Background()
	err := bus.Publish(ctx, NewInvalidationEvent(EntityService, "srv-1", ActionDeleted))
	if err == nil {
		t.Errorf("expected error when publishing to closed bus")
	}

	err = bus.Subscribe(ctx, func(event InvalidationEvent) {})
	if err == nil {
		t.Errorf("expected error when subscribing to closed bus")
	}
}

func TestInvalidationEventJSON(t *testing.T) {
	evt := NewInvalidationEvent(EntityConsumer, "user-456", ActionCreated)

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded InvalidationEvent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.EntityType != EntityConsumer || decoded.EntityID != "user-456" || decoded.Action != ActionCreated {
		t.Errorf("unexpected decoded values: %+v", decoded)
	}
	if decoded.Timestamp.IsZero() {
		t.Errorf("expected non-zero timestamp")
	}
}
