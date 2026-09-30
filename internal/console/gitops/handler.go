package gitops

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/apirout/apirout/internal/console/registry"
	"gopkg.in/yaml.v3"
)

// Handler handles GitOps declarative metadata export and import
type Handler struct {
	registry  *registry.Registry
	publisher *publisher.RedisPublisher
}

// NewHandler creates a new GitOps handler
func NewHandler(reg *registry.Registry, pub *publisher.RedisPublisher) *Handler {
	return &Handler{
		registry:  reg,
		publisher: pub,
	}
}

// Export exports complete declarative metadata snapshot in YAML format
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "SUPER_ADMIN" {
		httputil.WriteJSONError(w, http.StatusForbidden, "Only SUPER_ADMIN can export system metadata")
		return
	}

	groups, _ := h.registry.ListGroups()
	connections, _ := h.registry.ListConnections()
	queries, _ := h.registry.ListQueries()
	endpoints, _ := h.registry.ListEndpointsWithMetadata()

	groupMap := make(map[string]string)
	for _, g := range groups {
		groupMap[g.ID] = g.Name
	}

	connMap := make(map[string]string)
	for _, c := range connections {
		connMap[c.ID] = c.Name
	}

	queryMap := make(map[string]string)
	for _, q := range queries {
		queryMap[q.ID] = q.Name
	}

	snapshot := MetadataSnapshot{
		Version:    "api.apirout.io/v1",
		ExportedAt: time.Now(),
	}

	for _, g := range groups {
		snapshot.Groups = append(snapshot.Groups, GroupSpec{
			Name:        g.Name,
			Description: g.Description,
		})
	}

	for _, c := range connections {
		grpName := ""
		if c.GroupID != nil {
			grpName = groupMap[*c.GroupID]
		}
		snapshot.Connections = append(snapshot.Connections, ConnectionSpec{
			Name:      c.Name,
			DBType:    c.DBType,
			Host:      c.Host,
			Port:      c.Port,
			DBName:    c.DBName,
			Username:  c.Username,
			SSLMode:   c.SSLMode,
			GroupName: grpName,
		})
	}

	for _, q := range queries {
		grpName := ""
		if q.GroupID != nil {
			grpName = groupMap[*q.GroupID]
		}
		connName := ""
		if q.ConnectionID != nil {
			connName = connMap[*q.ConnectionID]
		}
		snapshot.Queries = append(snapshot.Queries, QuerySpec{
			Name:         q.Name,
			GroupName:    grpName,
			Connection:   connName,
			SQLText:      q.SQLText,
			OrderBy:      q.OrderBy,
			CountSQLText: q.CountSQLText,
			DefaultLimit: q.DefaultLimit,
		})
	}

	for _, e := range endpoints {
		grpName := ""
		if e.GroupID != nil {
			grpName = groupMap[*e.GroupID]
		}
		qName := ""
		if e.QueryID != nil {
			qName = queryMap[*e.QueryID]
		}
		svcName := ""
		if e.ServiceName != nil {
			svcName = *e.ServiceName
		}
		snapshot.Endpoints = append(snapshot.Endpoints, EndpointSpec{
			Path:                e.Path,
			Name:                e.Name,
			Method:              e.Method,
			ServiceName:         svcName,
			QueryName:           qName,
			GroupName:           grpName,
			Enabled:             e.Enabled,
			IsPublic:            e.IsPublic,
			Protocols:           e.Protocols,
			StripPath:           e.StripPath,
			PreserveHost:        e.PreserveHost,
			DefaultLimit:        e.DefaultLimit,
			MaxLimit:            e.MaxLimit,
			RateLimitRPM:        e.RateLimitRPM,
			RateLimitBurst:      e.RateLimitBurst,
			AllowDynamicFilters: e.AllowDynamicFilters,
			ReturnTotalCount:    e.ReturnTotalCount,
		})
	}

	// Export Services
	services, _ := h.registry.ListServices()
	for _, s := range services {
		upName := ""
		if s.UpstreamName != nil {
			upName = *s.UpstreamName
		}
		cName := ""
		if s.ConnectionName != nil {
			cName = *s.ConnectionName
		}
		snapshot.Services = append(snapshot.Services, ServiceSpec{
			Name:           s.Name,
			Description:    s.Description,
			Protocol:       s.Protocol,
			Host:           s.Host,
			Port:           s.Port,
			Path:           s.Path,
			ConnectTimeout: s.ConnectTimeout,
			ReadTimeout:    s.ReadTimeout,
			WriteTimeout:   s.WriteTimeout,
			Retries:        s.Retries,
			Enabled:        s.Enabled,
			UpstreamName:   upName,
			ConnectionName: cName,
		})
	}

	// Export Upstreams with Targets
	upstreams, _ := h.registry.ListUpstreams()
	for _, u := range upstreams {
		uSpec := UpstreamSpec{
			Name:                u.Name,
			Algorithm:           u.Algorithm,
			Slots:               u.Slots,
			Enabled:             u.Enabled,
			HealthCheckEnabled:  u.HealthCheckEnabled,
			HealthCheckPath:     u.HealthCheckPath,
			HealthCheckInterval: u.HealthCheckInterval,
			HealthCheckTimeout:  u.HealthCheckTimeout,
		}
		for _, t := range u.Targets {
			uSpec.Targets = append(uSpec.Targets, TargetSpec{
				Target:  t.Target,
				Host:    t.Host,
				Port:    t.Port,
				Weight:  t.Weight,
				Enabled: t.Enabled,
			})
		}
		snapshot.Upstreams = append(snapshot.Upstreams, uSpec)
	}

	// Export Consumers
	consumers, _ := h.registry.ListConsumers()
	for _, c := range consumers {
		grpName := ""
		if c.GroupName != nil {
			grpName = *c.GroupName
		}
		snapshot.Consumers = append(snapshot.Consumers, ConsumerSpec{
			Username:    c.Username,
			CustomID:    c.CustomID,
			Description: c.Description,
			GroupName:   grpName,
			Status:      c.Status,
		})
	}

	format := r.URL.Query().Get("format")
	if format == "json" {
		httputil.WriteJSON(w, http.StatusOK, snapshot)
		return
	}

	// Default to YAML
	yamlBytes, err := yaml.Marshal(snapshot)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to marshal YAML")
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=\"apirout-metadata.yaml\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(yamlBytes)
}

// Import imports declarative metadata snapshot
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "SUPER_ADMIN" {
		httputil.WriteJSONError(w, http.StatusForbidden, "Only SUPER_ADMIN can import system metadata")
		return
	}

	var snapshot MetadataSnapshot
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}
	} else {
		if err := yaml.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid YAML payload")
			return
		}
	}

	// Trigger broadcast to Data Plane to invalidate caches
	if h.publisher != nil {
		_ = h.publisher.PublishEndpointChanged("BULK_IMPORT", "", "")
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":  true,
		"message": "Metadata import parsed successfully",
		"stats": map[string]int{
			"groups":      len(snapshot.Groups),
			"services":    len(snapshot.Services),
			"upstreams":   len(snapshot.Upstreams),
			"consumers":   len(snapshot.Consumers),
			"connections": len(snapshot.Connections),
			"queries":     len(snapshot.Queries),
			"endpoints":   len(snapshot.Endpoints),
		},
	})
}
