package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/gorilla/mux"
)

// ConsumersHandler handles external Consumer and Consumer Group operations
type ConsumersHandler struct {
	registry *registry.Registry
}

// NewConsumersHandler creates a new ConsumersHandler
func NewConsumersHandler(reg *registry.Registry) *ConsumersHandler {
	return &ConsumersHandler{registry: reg}
}

// List returns all consumers
func (h *ConsumersHandler) List(w http.ResponseWriter, r *http.Request) {
	consumers, err := h.registry.ListConsumers()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list consumers: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, consumers)
}

// Get returns a single consumer with detail counts
func (h *ConsumersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	consumer, err := h.registry.GetConsumerByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get consumer: "+err.Error())
		return
	}
	if consumer == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Consumer not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, consumer)
}

// CreateConsumerRequest represents payload to create consumer
type CreateConsumerRequest struct {
	Username    string   `json:"username"`
	CustomID    *string  `json:"custom_id"`
	Description *string  `json:"description"`
	GroupID     *string  `json:"group_id"`
	Status      string   `json:"status"`
	Password    string   `json:"password"`
	EndpointIDs []string `json:"endpoint_ids"`
}

// Create creates a new external API consumer
func (h *ConsumersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateConsumerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Username is required")
		return
	}

	if req.Status == "" {
		req.Status = "active"
	}

	consumer := &models.Consumer{
		Username:    req.Username,
		CustomID:    req.CustomID,
		Description: req.Description,
		GroupID:     req.GroupID,
		Status:      req.Status,
	}

	created, err := h.registry.CreateConsumer(consumer, req.Password, req.EndpointIDs)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create consumer: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, created)
}

// Update updates consumer metadata
func (h *ConsumersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.registry.GetConsumerByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get consumer: "+err.Error())
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Consumer not found")
		return
	}

	var req CreateConsumerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CustomID != nil {
		existing.CustomID = req.CustomID
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.GroupID != nil {
		existing.GroupID = req.GroupID
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	c := existing.ToConsumer()
	if err := h.registry.UpdateConsumer(c, req.EndpointIDs); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update consumer: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// Delete deletes a consumer
func (h *ConsumersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.registry.DeleteAPIClient(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete consumer: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Consumer deleted successfully"})
}

// ==========================================
// CREDENTIALS MANAGEMENT
// ==========================================

// ListCredentials returns masked credentials for a consumer
func (h *ConsumersHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	consumerID := mux.Vars(r)["id"]
	creds, err := h.registry.ListConsumerCredentials(consumerID)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list credentials: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, creds)
}

// CreateCredentialRequest represents request to generate API key
type CreateCredentialRequest struct {
	Name      string     `json:"name"`
	KeyType   string     `json:"key_type"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// CreateCredential generates a new API Key credential (revealing secret once)
func (h *ConsumersHandler) CreateCredential(w http.ResponseWriter, r *http.Request) {
	consumerID := mux.Vars(r)["id"]
	var req CreateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		req.Name = "API Key"
	}
	if req.KeyType == "" {
		req.KeyType = "api-key"
	}

	created, err := h.registry.CreateConsumerCredential(consumerID, req.Name, req.KeyType, req.ExpiresAt)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create credential: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, created)
}

// ToggleCredential toggles enable/disable
func (h *ConsumersHandler) ToggleCredential(w http.ResponseWriter, r *http.Request) {
	credID := mux.Vars(r)["credId"]
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.registry.ToggleConsumerCredential(credID, req.Enabled); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to toggle credential: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"id": credID, "enabled": req.Enabled})
}

// DeleteCredential revokes a credential
func (h *ConsumersHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	credID := mux.Vars(r)["credId"]
	if err := h.registry.DeleteConsumerCredential(credID); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to revoke credential: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Credential revoked successfully"})
}

// GetRoutes returns the allowed routes for a consumer
func (h *ConsumersHandler) GetRoutes(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	routeIDs, err := h.registry.GetAPIClientAllowedEndpointIDs(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get allowed routes")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"route_ids": routeIDs})
}

// UpdateRoutes updates the allowed routes for a consumer
func (h *ConsumersHandler) UpdateRoutes(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req struct {
		RouteIDs []string `json:"route_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	consumer, err := h.registry.GetConsumerByID(id)
	if err != nil || consumer == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Consumer not found")
		return
	}

	c := consumer.ToConsumer()
	if err := h.registry.UpdateConsumer(c, req.RouteIDs); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update routes: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{"message": "Routes updated successfully", "route_ids": req.RouteIDs})
}

// ==========================================
// CONSUMER GROUPS ENDPOINTS
// ==========================================

// ListGroups returns all consumer groups
func (h *ConsumersHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.registry.ListConsumerGroups()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list consumer groups: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, groups)
}

// GetGroup returns a single consumer group
func (h *ConsumersHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	group, err := h.registry.GetConsumerGroupByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get consumer group: "+err.Error())
		return
	}
	if group == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Consumer group not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, group)
}

// CreateGroup creates a new consumer group
func (h *ConsumersHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string  `json:"name"`
		Description    *string `json:"description"`
		RateLimitRPM   int     `json:"rate_limit_rpm"`
		RateLimitBurst int     `json:"rate_limit_burst"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Group name is required")
		return
	}

	group := &models.ConsumerGroup{
		Name:           req.Name,
		Description:    req.Description,
		RateLimitRPM:   req.RateLimitRPM,
		RateLimitBurst: req.RateLimitBurst,
	}

	if err := h.registry.CreateConsumerGroup(group); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create consumer group: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, group)
}

// UpdateGroup updates a consumer group
func (h *ConsumersHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.registry.GetConsumerGroupByID(id)
	if err != nil || existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Consumer group not found")
		return
	}

	var req struct {
		Name           string  `json:"name"`
		Description    *string `json:"description"`
		RateLimitRPM   int     `json:"rate_limit_rpm"`
		RateLimitBurst int     `json:"rate_limit_burst"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Description = req.Description
	if req.RateLimitRPM > 0 {
		existing.RateLimitRPM = req.RateLimitRPM
	}
	if req.RateLimitBurst > 0 {
		existing.RateLimitBurst = req.RateLimitBurst
	}

	if err := h.registry.UpdateConsumerGroup(existing); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update consumer group: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// DeleteGroup deletes a consumer group
func (h *ConsumersHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.registry.DeleteConsumerGroup(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete consumer group: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Consumer group deleted successfully"})
}
