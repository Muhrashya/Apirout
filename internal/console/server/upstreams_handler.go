package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/gorilla/mux"
)

// UpstreamsHandler handles Upstreams and Targets management
type UpstreamsHandler struct {
	registry *registry.Registry
}

// NewUpstreamsHandler creates a new UpstreamsHandler
func NewUpstreamsHandler(reg *registry.Registry) *UpstreamsHandler {
	return &UpstreamsHandler{registry: reg}
}

// List returns all upstreams with target counts
func (h *UpstreamsHandler) List(w http.ResponseWriter, r *http.Request) {
	upstreams, err := h.registry.ListUpstreams()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list upstreams: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, upstreams)
}

// Get returns a single upstream by ID with targets
func (h *UpstreamsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	upstream, err := h.registry.GetUpstreamByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get upstream: "+err.Error())
		return
	}
	if upstream == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Upstream not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, upstream)
}

// CreateUpstreamRequest represents creation payload
type CreateUpstreamRequest struct {
	Name                string  `json:"name"`
	Algorithm           string  `json:"algorithm"`
	HashOn              string  `json:"hash_on"`
	HashFallback        string  `json:"hash_fallback"`
	HashOnCookie        *string `json:"hash_on_cookie"`
	Slots               int     `json:"slots"`
	Enabled             bool    `json:"enabled"`
	HealthCheckEnabled  bool    `json:"health_check_enabled"`
	HealthCheckPath     string  `json:"health_check_path"`
	HealthCheckInterval int     `json:"health_check_interval"`
	HealthCheckTimeout  int     `json:"health_check_timeout"`
	HealthyThreshold    int     `json:"healthy_threshold"`
	UnhealthyThreshold  int     `json:"unhealthy_threshold"`
}

// Create creates a new Upstream pool
func (h *UpstreamsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Upstream name is required")
		return
	}

	upstream := &models.Upstream{
		Name:                req.Name,
		Algorithm:           req.Algorithm,
		HashOn:              req.HashOn,
		HashFallback:        req.HashFallback,
		HashOnCookie:        req.HashOnCookie,
		Slots:               req.Slots,
		Enabled:             req.Enabled,
		HealthCheckEnabled:  req.HealthCheckEnabled,
		HealthCheckPath:     req.HealthCheckPath,
		HealthCheckInterval: req.HealthCheckInterval,
		HealthCheckTimeout:  req.HealthCheckTimeout,
		HealthyThreshold:    req.HealthyThreshold,
		UnhealthyThreshold:  req.UnhealthyThreshold,
	}

	if err := h.registry.CreateUpstream(upstream); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create upstream: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, upstream)
}

// Update updates an existing upstream pool
func (h *UpstreamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.registry.GetUpstreamByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get upstream: "+err.Error())
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Upstream not found")
		return
	}

	var req CreateUpstreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Algorithm != "" {
		existing.Algorithm = req.Algorithm
	}
	if req.HashOn != "" {
		existing.HashOn = req.HashOn
	}
	existing.HashFallback = req.HashFallback
	existing.HashOnCookie = req.HashOnCookie
	if req.Slots > 0 {
		existing.Slots = req.Slots
	}
	existing.Enabled = req.Enabled
	existing.HealthCheckEnabled = req.HealthCheckEnabled
	if req.HealthCheckPath != "" {
		existing.HealthCheckPath = req.HealthCheckPath
	}
	if req.HealthCheckInterval > 0 {
		existing.HealthCheckInterval = req.HealthCheckInterval
	}
	if req.HealthCheckTimeout > 0 {
		existing.HealthCheckTimeout = req.HealthCheckTimeout
	}
	if req.HealthyThreshold > 0 {
		existing.HealthyThreshold = req.HealthyThreshold
	}
	if req.UnhealthyThreshold > 0 {
		existing.UnhealthyThreshold = req.UnhealthyThreshold
	}

	up := existing.ToUpstream()
	if err := h.registry.UpdateUpstream(up); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update upstream: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// Delete deletes an upstream pool
func (h *UpstreamsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.registry.DeleteUpstream(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete upstream: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Upstream deleted successfully"})
}

// ==========================================
// TARGETS ENDPOINTS
// ==========================================

// ListTargets returns targets for a specific upstream
func (h *UpstreamsHandler) ListTargets(w http.ResponseWriter, r *http.Request) {
	upstreamID := mux.Vars(r)["id"]
	targets, err := h.registry.ListTargetsByUpstreamID(upstreamID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list targets: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, targets)
}

// CreateTargetRequest represents create target payload
type CreateTargetRequest struct {
	Target   string `json:"target"` // host:port
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
	Failover bool   `json:"failover"`
}

// CreateTarget creates a new backend instance target
func (h *UpstreamsHandler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	upstreamID := mux.Vars(r)["id"]
	var req CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Target == "" && req.Host == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Target address or host is required")
		return
	}

	if req.Weight <= 0 {
		req.Weight = 100
	}

	target := &models.Target{
		UpstreamID:   upstreamID,
		Target:       req.Target,
		Host:         req.Host,
		Port:         req.Port,
		Weight:       req.Weight,
		Enabled:      req.Enabled,
		HealthStatus: "healthy",
		Failover:     req.Failover,
	}

	if err := h.registry.CreateTarget(target); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create target: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, target)
}

// UpdateTarget updates weight, failover or host of a target
func (h *UpstreamsHandler) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["targetId"]
	existing, err := h.registry.GetTargetByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get target: "+err.Error())
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Target not found")
		return
	}

	var req CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Target != "" {
		existing.Target = req.Target
	}
	if req.Host != "" {
		existing.Host = req.Host
	}
	if req.Port > 0 {
		existing.Port = req.Port
	}
	if req.Weight > 0 {
		existing.Weight = req.Weight
	}
	existing.Enabled = req.Enabled
	existing.Failover = req.Failover

	if err := h.registry.UpdateTarget(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update target: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// DeleteTarget deletes a target
func (h *UpstreamsHandler) DeleteTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["targetId"]
	if err := h.registry.DeleteTarget(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete target: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Target deleted successfully"})
}

// ToggleTarget enables or disables a target
func (h *UpstreamsHandler) ToggleTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["targetId"]
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	status := "healthy"
	if !req.Enabled {
		status = "disabled"
	}

	if err := h.registry.UpdateTargetEnabled(id, req.Enabled); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to toggle target: "+err.Error())
		return
	}
	_ = h.registry.UpdateTargetHealthStatus(id, status)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"id": id, "enabled": req.Enabled, "health_status": status})
}

// HealthCheckTarget executes an immediate active health check for a single target
func (h *UpstreamsHandler) HealthCheckTarget(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["targetId"]
	target, err := h.registry.GetTargetByID(id)
	if err != nil || target == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Target not found")
		return
	}

	upstream, err := h.registry.GetUpstreamByID(target.UpstreamID)
	healthPath := "/health"
	timeoutSec := 5
	if upstream != nil {
		if upstream.HealthCheckPath != "" {
			healthPath = upstream.HealthCheckPath
		}
		if upstream.HealthCheckTimeout > 0 {
			timeoutSec = upstream.HealthCheckTimeout
		}
	}

	status, latency, errMsg := performActiveHealthCheck(target.Host, target.Port, healthPath, timeoutSec)
	_ = h.registry.UpdateTargetHealthStatus(target.ID, status)

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"target_id":     target.ID,
		"target":        target.Target,
		"health_status": status,
		"latency_ms":    latency,
		"error":         errMsg,
		"checked_at":    time.Now().Format(time.RFC3339),
	})
}

// HealthCheckUpstream runs active health checks across all targets of an upstream
func (h *UpstreamsHandler) HealthCheckUpstream(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	upstream, err := h.registry.GetUpstreamByID(id)
	if err != nil || upstream == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Upstream not found")
		return
	}

	targets, err := h.registry.ListTargetsByUpstreamID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list targets")
		return
	}

	results := make([]map[string]interface{}, 0, len(targets))
	for _, target := range targets {
		if !target.Enabled {
			results = append(results, map[string]interface{}{
				"target_id":     target.ID,
				"target":        target.Target,
				"health_status": "disabled",
			})
			continue
		}

		status, latency, errMsg := performActiveHealthCheck(target.Host, target.Port, upstream.HealthCheckPath, upstream.HealthCheckTimeout)
		_ = h.registry.UpdateTargetHealthStatus(target.ID, status)

		results = append(results, map[string]interface{}{
			"target_id":     target.ID,
			"target":        target.Target,
			"health_status": status,
			"latency_ms":    latency,
			"error":         errMsg,
		})
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"upstream_id": upstream.ID,
		"name":        upstream.Name,
		"results":     results,
		"checked_at":  time.Now().Format(time.RFC3339),
	})
}

// performActiveHealthCheck actually performs an HTTP health check request
func performActiveHealthCheck(host string, port int, path string, timeoutSec int) (status string, latencyMs int64, errMsg string) {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	if path == "" {
		path = "/health"
	}
	if path[0] != '/' {
		path = "/" + path
	}

	url := fmt.Sprintf("http://%s:%d%s", host, port, path)
	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return "unhealthy", latency, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return "healthy", latency, ""
	}

	return "unhealthy", latency, fmt.Sprintf("HTTP status %d", resp.StatusCode)
}
