package monitoring

import (
	"net/http"
	"strconv"

	"github.com/apirout/apirout/internal/console/registry"
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

// AuditLogs returns audit logs with pagination and filtering
func (h *DashboardHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	limit := 25
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}

	eventType := r.URL.Query().Get("event_type")
	search := r.URL.Query().Get("search")

	logs, total, err := h.service.GetAuditLogsPaginated(eventType, search, page, limit)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get audit logs: "+err.Error())
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": logs,
		"pagination": map[string]interface{}{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// Requests returns API request logs for the Kong-aligned request explorer with pagination and filtering
func (h *DashboardHandler) Requests(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}

	limit := 25
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if limit > 100 {
		limit = 100
	}

	filter := registry.RequestFilter{
		Search:      r.URL.Query().Get("search"),
		Method:      r.URL.Query().Get("method"),
		StatusGroup: r.URL.Query().Get("status"),
	}

	reqs, total, err := h.service.GetRequestsPaginated(filter, page, limit)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get request logs: "+err.Error())
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": reqs,
		"pagination": map[string]interface{}{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}
