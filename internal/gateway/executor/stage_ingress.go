package executor

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/core/iputil"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/router"
	"github.com/apirout/apirout/internal/gateway/pipeline"
	"github.com/apirout/apirout/internal/gateway/responder"
)

type ingressStage struct {
	handler *Handler
}

func newIngressStage(h *Handler) *ingressStage {
	return &ingressStage{handler: h}
}

func (s *ingressStage) Name() string { return "ingress" }

func (s *ingressStage) Execute(c *pipeline.Context) error {
	r := c.Request
	rec := c.Recorder

	if !strings.HasPrefix(c.Path, "/api/") {
		c.SetError(http.StatusNotFound, "Not found")
		responder.Error(rec, r, http.StatusNotFound, responder.ErrCodeRouteNotFound, "Not found")
		return nil
	}

	// 1. In-Memory Radix Route Matcher (<1ms)
	var endpoint *models.Endpoint
	var pathParams map[string]string
	curRouter := s.handler.router.Load()
	if curRouter != nil {
		res, matchErr := curRouter.Match(c.Path, r.Method)
		if matchErr == nil && res != nil {
			endpoint = res.Endpoint
			pathParams = res.Params
		} else if errors.Is(matchErr, router.ErrMethodNotAllowed) {
			c.SetError(http.StatusMethodNotAllowed, "Method not allowed")
			extra := make(map[string]string)
			if len(res.AllowedMethods) > 0 {
				extra["Allow"] = strings.Join(res.AllowedMethods, ", ")
			}
			responder.Error(rec, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Method not allowed", extra)
			return nil
		}
	}

	// Fallback to registry reader if route not found in in-memory tree
	if endpoint == nil && s.handler.reader != nil {
		ep, err := s.handler.reader.GetEndpointByPath(c.Path)
		if err != nil {
			c.SetError(http.StatusInternalServerError, "Database error")
			responder.Error(rec, r, http.StatusInternalServerError, responder.ErrCodeInternalError, "Database error")
			return nil
		}
		endpoint = ep
	}

	if endpoint == nil {
		c.SetError(http.StatusNotFound, "Endpoint not found")
		responder.Error(rec, r, http.StatusNotFound, responder.ErrCodeRouteNotFound, "Endpoint not found")
		return nil
	}

	c.Endpoint = endpoint
	c.PathParams = pathParams
	c.RouteName = endpoint.Name

	if !endpoint.Enabled {
		c.SetError(http.StatusNotFound, "Endpoint disabled")
		responder.Error(rec, r, http.StatusNotFound, responder.ErrCodeRouteNotFound, "Endpoint disabled")
		return nil
	}

	// Match HTTP method (for fallback paths)
	if endpoint.Method != "*" && endpoint.Method != "ALL" && endpoint.Method != "ANY" && endpoint.Method != "" && !strings.Contains(endpoint.Method, r.Method) {
		c.SetError(http.StatusMethodNotAllowed, "Method not allowed")
		responder.Error(rec, r, http.StatusMethodNotAllowed, responder.ErrCodeMethodNotAllowed, "Method not allowed", map[string]string{"Allow": endpoint.Method})
		return nil
	}

	// Host-based / Domain Routing check
	host := r.Host
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		host = xfh
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// 1. Gateway-level host validation against registered gateway domains
	if !s.handler.isGatewayHostAllowed(host) {
		errStr := fmt.Sprintf("Host '%s' is not registered on this gateway", host)
		c.SetError(http.StatusForbidden, errStr)
		responder.Error(rec, r, http.StatusForbidden, responder.ErrCodeForbidden, errStr)
		return nil
	}

	// 2. Endpoint-level domain restriction check
	if !isDomainAllowed(host, endpoint.AllowedDomains) {
		c.SetError(http.StatusForbidden, "Domain not authorized to access this endpoint")
		responder.Error(rec, r, http.StatusForbidden, responder.ErrCodeForbidden, "Domain not authorized to access this endpoint")
		return nil
	}

	// Dynamic Route-Level CORS handling
	reqOrigin := r.Header.Get("Origin")
	if reqOrigin != "" && endpoint.CorsOrigins != "" && endpoint.CorsOrigins != "*" {
		if isOriginAllowed(reqOrigin, endpoint.CorsOrigins) {
			rec.Header().Set("Access-Control-Allow-Origin", reqOrigin)
			rec.Header().Set("Vary", "Origin")
			rec.Header().Set("Access-Control-Allow-Credentials", "true")
		}
	} else if endpoint.CorsOrigins == "*" {
		rec.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// Handle preflight OPTIONS request for endpoint
	if r.Method == http.MethodOptions {
		rec.Header().Set("Access-Control-Allow-Methods", endpoint.Method)
		rec.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Admin-Secret, X-Request-ID, X-Correlation-ID")
		rec.Header().Set("Access-Control-Max-Age", "86400")
		rec.WriteHeader(http.StatusNoContent)
		c.MarkHandled()
		return nil
	}

	// Client IP Whitelist Check (CIDR / direct IP / wildcard)
	if !iputil.IsIPAllowed(c.ClientIP, endpoint.AllowedIPs) {
		c.SetError(http.StatusForbidden, "Client IP address forbidden for this endpoint")
		responder.Error(rec, r, http.StatusForbidden, responder.ErrCodeForbidden, "Client IP address forbidden for this endpoint")
		return nil
	}

	return nil
}
