package eventbus

import (
	"time"
)

// EntityType represents the domain category being invalidated
type EntityType string

const (
	EntityRoute         EntityType = "route"
	EntityService       EntityType = "service"
	EntityUpstream      EntityType = "upstream"
	EntityConsumer      EntityType = "consumer"
	EntityConsumerGroup EntityType = "consumer_group"
	EntityAuthPlugin    EntityType = "auth_plugin"
	EntityConnection    EntityType = "connection"
	EntityQuery         EntityType = "query"
	EntitySetting       EntityType = "setting"
	EntityAll           EntityType = "*"
)

// ActionType represents the mutation action
type ActionType string

const (
	ActionCreated ActionType = "created"
	ActionUpdated ActionType = "updated"
	ActionDeleted ActionType = "deleted"
	ActionPurged  ActionType = "purged"
)

// InvalidationEvent is the standard event envelope for cross-plane synchronization
type InvalidationEvent struct {
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"` // UUID, name, or "*" for bulk
	Action     ActionType `json:"action"`
	Timestamp  time.Time  `json:"timestamp"`
}

// NewInvalidationEvent creates a timestamped invalidation event
func NewInvalidationEvent(entityType EntityType, entityID string, action ActionType) InvalidationEvent {
	return InvalidationEvent{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Timestamp:  time.Now().UTC(),
	}
}
