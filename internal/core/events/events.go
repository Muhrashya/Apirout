package events

import (
	"encoding/json"
	"time"
)

const (
	// ChannelConfigEvents is the Redis Pub/Sub channel for endpoint and query changes
	ChannelConfigEvents = "apirout:events:config"
	// ChannelConnectionEvents is the Redis Pub/Sub channel for target database connection changes
	ChannelConnectionEvents = "apirout:events:connection"

	ActionCreate  = "CREATE"
	ActionUpdate  = "UPDATE"
	ActionDelete  = "DELETE"
	ActionEnable  = "ENABLE"
	ActionDisable = "DISABLE"
)

// EndpointInvalidateEvent is published by Control Plane when an endpoint is changed
type EndpointInvalidateEvent struct {
	Action     string `json:"action"`
	EndpointID string `json:"endpoint_id"`
	Path       string `json:"path"`
	Timestamp  int64  `json:"timestamp"`
}

// NewEndpointInvalidateEvent creates an event payload
func NewEndpointInvalidateEvent(action, endpointID, path string) *EndpointInvalidateEvent {
	return &EndpointInvalidateEvent{
		Action:     action,
		EndpointID: endpointID,
		Path:       path,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// Encode converts event to JSON bytes
func (e *EndpointInvalidateEvent) Encode() ([]byte, error) {
	return json.Marshal(e)
}

// DecodeEndpointInvalidateEvent parses JSON bytes into event
func DecodeEndpointInvalidateEvent(data []byte) (*EndpointInvalidateEvent, error) {
	var e EndpointInvalidateEvent
	err := json.Unmarshal(data, &e)
	return &e, err
}

// ConnectionInvalidateEvent is published when database connection credentials change
type ConnectionInvalidateEvent struct {
	Action       string `json:"action"`
	ConnectionID string `json:"connection_id"`
	Timestamp    int64  `json:"timestamp"`
}

// NewConnectionInvalidateEvent creates a connection invalidate event
func NewConnectionInvalidateEvent(action, connectionID string) *ConnectionInvalidateEvent {
	return &ConnectionInvalidateEvent{
		Action:       action,
		ConnectionID: connectionID,
		Timestamp:    time.Now().UnixMilli(),
	}
}

// Encode converts event to JSON bytes
func (e *ConnectionInvalidateEvent) Encode() ([]byte, error) {
	return json.Marshal(e)
}

// DecodeConnectionInvalidateEvent parses JSON bytes into event
func DecodeConnectionInvalidateEvent(data []byte) (*ConnectionInvalidateEvent, error) {
	var e ConnectionInvalidateEvent
	err := json.Unmarshal(data, &e)
	return &e, err
}
