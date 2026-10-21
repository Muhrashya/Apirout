package executor

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/pipeline"
	"github.com/apirout/apirout/internal/gateway/responder"
)

type upstreamStage struct {
	handler *Handler
}

func newUpstreamStage(h *Handler) *upstreamStage {
	return &upstreamStage{handler: h}
}

func (s *upstreamStage) Name() string { return "upstream" }

func (s *upstreamStage) Execute(c *pipeline.Context) error {
	endpoint := c.Endpoint
	if endpoint == nil {
		return nil
	}

	r := c.Request
	rec := c.Recorder
	h := s.handler

	var session *SessionContext
	if c.ConsumerID != nil {
		session = &SessionContext{
			SubjectID:   *c.ConsumerID,
			SubjectType: "api_client",
			APIClientID: *c.ConsumerID,
			GroupID:     c.GroupID,
		}
	}

	// Case Instant-API: Direct /api/v1/instant/{connId}/* Unified Database Pipeline
	if strings.HasPrefix(c.Path, "/api/v1/instant/") && h.instantHandler != nil {
		parts := strings.Split(strings.Trim(c.Path, "/"), "/")
		if len(parts) >= 4 {
			connID := parts[3]
			c.ServiceName = "Instant Database Service"
			c.UpstreamURI = fmt.Sprintf("database://%s", connID)

			// 1. Authorize connection access (handles Admin Secret, API Key, JWT token, tenant group check)
			authStatus, authErr := h.instantHandler.AuthorizeConnectionAccess(r, connID)
			if authErr != nil {
				c.SetError(authStatus, authErr.Error())
				if authStatus == http.StatusForbidden {
					responder.Error(rec, r, http.StatusForbidden, responder.ErrCodeForbidden, authErr.Error())
				} else if authStatus == http.StatusNotFound {
					responder.Error(rec, r, http.StatusNotFound, responder.ErrCodeNotFound, authErr.Error())
				} else {
					responder.Error(rec, r, http.StatusUnauthorized, responder.ErrCodeUnauthorized, authErr.Error())
				}
				c.MarkHandled()
				return nil
			}

			// 2. Check read-only connection enforcement before mutating table CRUD
			if h.reader != nil {
				conn, err := h.reader.GetConnectionByID(connID)
				if err == nil && conn != nil && conn.IsReadOnly {
					if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
						rec.Header().Set("Allow", "GET, HEAD, OPTIONS")
						c.SetError(http.StatusMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited")
						responder.Error(rec, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
						c.MarkHandled()
						return nil
					}
				}
			}

			// 3. Dispatch to schema or table CRUD
			upStart := time.Now()
			if len(parts) >= 5 && parts[4] == "schema" {
				vars := map[string]string{"connId": connID}
				r = mux.SetURLVars(r, vars)
				h.instantHandler.GetSchema(rec, r)
			} else if len(parts) >= 5 {
				tableName := parts[4]
				id := ""
				if len(parts) >= 6 {
					id = parts[5]
				}
				h.instantHandler.ExecuteTableCRUD(rec, r, connID, tableName, id)
			} else {
				c.SetError(http.StatusBadRequest, "Missing table or schema specification in instant path")
				responder.Error(rec, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Missing table or schema specification in instant path")
				c.MarkHandled()
				return nil
			}

			c.UpstreamDurationMs = int(time.Since(upStart).Milliseconds())
			c.StatusCode = rec.StatusCode
			if c.StatusCode >= 400 {
				c.Success = false
			}
			c.MarkHandled()
			return nil
		}
	}

	// 3. Service Resolution & Gateway Execution
	var outboundAuth *models.ServiceOutboundAuth
	if endpoint.ServiceID != nil && *endpoint.ServiceID != "" && h.reader != nil {
		service, err := h.reader.GetServiceByID(*endpoint.ServiceID)
		if err == nil && service != nil && service.Enabled {
			c.ServiceName = service.Name

			// Load outbound auth config for this service
			outboundAuth, _ = h.reader.GetOutboundAuth(service.ID)

			// Case A: Service routes to an Upstream pool (Load-balanced Targets)
			if service.UpstreamID != nil && *service.UpstreamID != "" {
				upstream, _ := h.reader.GetUpstreamByID(*service.UpstreamID)
				targets, err := h.reader.GetHealthyTargetsByUpstreamID(*service.UpstreamID)
				availableTargets := targets
				if h.circuitBreaker != nil {
					avail, _ := h.circuitBreaker.FilterAvailable(targets)
					availableTargets = avail
				}
				if err != nil || len(availableTargets) == 0 {
					c.SetError(http.StatusServiceUnavailable, "no healthy upstream targets available")
					responder.Error(rec, r, http.StatusServiceUnavailable, responder.ErrCodeNoHealthyTargets, "no healthy upstream targets available")
					return nil
				}

				target, releaseConn, err := h.targetPool.SelectTarget(r, upstream, availableTargets)
				if err != nil || target == nil {
					c.SetError(http.StatusServiceUnavailable, "no healthy upstream targets available")
					responder.Error(rec, r, http.StatusServiceUnavailable, responder.ErrCodeNoHealthyTargets, "no healthy upstream targets available")
					return nil
				}
				defer releaseConn()
				c.UpstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, target.Host, target.Port, service.Path)
				h.reverseProxyRequest(rec, r, endpoint, service, target.Host, target.Port, target.ID, outboundAuth, &c.UpstreamDurationMs)
				c.MarkHandled()
				return nil
			}

			// Case B: Service points directly to a backend host/port
			if service.Host != "" && service.Protocol != "database" {
				syntheticTargetID := "direct:" + service.ID
				if h.circuitBreaker != nil && !h.circuitBreaker.Allow(syntheticTargetID) {
					c.SetError(http.StatusServiceUnavailable, "service backend temporarily unavailable (circuit breaker open)")
					responder.Error(rec, r, http.StatusServiceUnavailable, responder.ErrCodeNoHealthyTargets, "service backend temporarily unavailable (circuit breaker open)")
					c.MarkHandled()
					return nil
				}

				c.UpstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, service.Host, service.Port, service.Path)
				h.reverseProxyRequest(rec, r, endpoint, service, service.Host, service.Port, syntheticTargetID, outboundAuth, &c.UpstreamDurationMs)
				c.MarkHandled()
				return nil
			}

			// Case C: Service routes to a Database Connection (Instant REST CRUD or SQL API)
			if (service.ConnectionID != nil && *service.ConnectionID != "") || service.Protocol == "database" {
				connID := ""
				if service.ConnectionID != nil {
					connID = *service.ConnectionID
				}
				if connID == "" {
					connID = service.ID
				}
				c.UpstreamURI = fmt.Sprintf("database://%s/%s", connID, service.Path)

				// C1: If route has an explicit SQL query attached, execute that query
				if endpoint.QueryID != nil && *endpoint.QueryID != "" {
					params := parseParams(r)
					for k, v := range c.PathParams {
						params[k] = v
					}
					upStart := time.Now()
					result, err := h.executor.ExecuteEndpointByPath(c.Path, params, session)
					c.UpstreamDurationMs = int(time.Since(upStart).Milliseconds())
					if err != nil {
						c.SetError(http.StatusInternalServerError, err.Error())
						responder.Error(rec, r, http.StatusInternalServerError, responder.ErrCodeInternalError, err.Error())
						return nil
					}
					responder.JSON(rec, r, http.StatusOK, result)
					c.MarkHandled()
					return nil
				}

				// C2: Route performs Instant REST table CRUD on the connected database
				if h.instantHandler != nil {
					// Check read-only connection enforcement before mutating table CRUD
					if h.reader != nil && connID != "" {
						conn, err := h.reader.GetConnectionByID(connID)
						if err == nil && conn != nil && conn.IsReadOnly {
							if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
								rec.Header().Set("Allow", "GET, HEAD, OPTIONS")
								c.SetError(http.StatusMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited")
								responder.Error(rec, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Connection is configured as read-only: mutating operations are prohibited", map[string]string{"Allow": "GET, HEAD, OPTIONS"})
								c.MarkHandled()
								return nil
							}
						}
					}

					table, id := extractTableAndID(endpoint.Path, r.URL.Path, service.Path)
					upStart := time.Now()
					h.instantHandler.ExecuteTableCRUD(rec, r, connID, table, id)
					c.UpstreamDurationMs = int(time.Since(upStart).Milliseconds())
					c.StatusCode = rec.StatusCode
					if c.StatusCode >= 400 {
						c.Success = false
					}
					c.MarkHandled()
					return nil
				}
			}
		}
	}

	// Case D: Endpoint with QueryID but no Service (Standalone SQL Query)
	if endpoint.QueryID != nil && *endpoint.QueryID != "" {
		if c.ServiceName == "" {
			c.ServiceName = "Standalone SQL Query"
			c.UpstreamURI = "internal://sql-engine"
		}
		params := parseParams(r)
		for k, v := range c.PathParams {
			params[k] = v
		}
		upStart := time.Now()
		result, err := h.executor.ExecuteEndpointByPath(c.Path, params, session)
		c.UpstreamDurationMs = int(time.Since(upStart).Milliseconds())
		if err != nil {
			c.SetError(http.StatusInternalServerError, err.Error())
			responder.Error(rec, r, http.StatusInternalServerError, responder.ErrCodeInternalError, err.Error())
			return nil
		}
		responder.JSON(rec, r, http.StatusOK, result)
		c.MarkHandled()
		return nil
	}

	// Case E: Route has neither a valid Service nor a Query
	c.SetError(http.StatusBadRequest, "Route is not configured with an active Service backend, Database connection, or SQL query")
	responder.Error(rec, r, http.StatusBadRequest, responder.ErrCodeBadRequest, "Route is not configured with an active Service backend, Database connection, or SQL query")
	return nil
}
