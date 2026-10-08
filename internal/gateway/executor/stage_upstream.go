package executor

import (
	"fmt"
	"net/http"
	"time"

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

				algorithm := "round-robin"
				if upstream != nil && upstream.Algorithm != "" {
					algorithm = upstream.Algorithm
				}
				target := selectTarget(availableTargets, algorithm, c.ClientIP, r, *service.UpstreamID)
				if target != nil {
					c.UpstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, target.Host, target.Port, service.Path)
					releaseConn := trackTargetConn(target.ID)
					defer releaseConn()
					h.reverseProxyRequest(rec, r, endpoint, service, target.Host, target.Port, target.ID, outboundAuth, &c.UpstreamDurationMs)
					c.MarkHandled()
					return nil
				}
			}

			// Case B: Service points directly to a backend host/port
			if service.Host != "" && service.Protocol != "database" {
				c.UpstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, service.Host, service.Port, service.Path)
				h.reverseProxyRequest(rec, r, endpoint, service, service.Host, service.Port, "", outboundAuth, &c.UpstreamDurationMs)
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
