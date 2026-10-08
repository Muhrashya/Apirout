# 52: Control Plane EventBus Publisher Binding for Route Mutations

**What to build:** Injects the distributed `publisher *publisher.RedisPublisher` into `EndpointHandler` in `internal/console/server/endpoints.go` and wires it in `internal/console/app/routes.go`. Broadcasts `PublishEndpointChanged` across Redis Pub/Sub whenever an endpoint is created, updated, or deleted, triggering dynamic in-memory Radix tree compilation and atomic pointer swapping on the Data Plane.

**Blocked by:** None (can start immediately)

**Status:** completed

- [x] `publisher *publisher.RedisPublisher` added to `EndpointHandler` struct and constructor
- [x] `internal/console/app/routes.go` passes `d.publisher` to `server.NewEndpointHandler`
- [x] `EndpointHandler.Create` emits `PublishEndpointChanged("CREATE", ...)`
- [x] `EndpointHandler.Update` emits `PublishEndpointChanged("UPDATE", ...)`
- [x] `EndpointHandler.Delete` emits `PublishEndpointChanged("DELETE", ...)`
- [x] Unit tests pass with `go test -v ./internal/...`
