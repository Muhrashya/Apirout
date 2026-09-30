package monitoring

import (
	"net/http"
	"strconv"

	"github.com/apirout/apirout/internal/core/httputil"
)

// DashboardHandler handles dashboard HTTP requests
type DashboardHandler struct {
	service *Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(service *Service) *DashboardHandler {
	return &DashboardHandler{service: service}
}

// Stats returns dashboard statistics
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetDashboardStats()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, stats)
}

// AuditLogs returns audit logs
func (h *DashboardHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("event_type")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.service.GetAuditLogs(eventType, limit)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get audit logs: "+err.Error())
		return
	}

	httputil.WriteJSONData(w, http.StatusOK, logs)
}

// Requests returns API request logs for the Kong-aligned request explorer
func (h *DashboardHandler) Requests(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}

	reqs, err := h.service.GetRequests(limit)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get request logs: "+err.Error())
		return
	}

	httputil.WriteJSONData(w, http.StatusOK, reqs)
}
