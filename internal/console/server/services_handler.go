package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/egress"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/gorilla/mux"
)

// ServicesHandler handles Gateway Service operations
type ServicesHandler struct {
	registry *registry.Registry
}

// NewServicesHandler creates a new ServicesHandler
func NewServicesHandler(reg *registry.Registry) *ServicesHandler {
	return &ServicesHandler{registry: reg}
}

func serviceEgressPolicy() egress.Policy {
	policy := egress.Policy{AllowPrivate: true}
	if os.Getenv("ALLOW_LOOPBACK_SERVICES") == "true" || os.Getenv("APP_ENV") != "production" {
		policy.AllowLoopback = true
	}
	return policy
}

// List returns all services
func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.registry.ListServices()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list services: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, services)
}

// Get returns a single service by ID
func (h *ServicesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	service, err := h.registry.GetServiceByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get service: "+err.Error())
		return
	}
	if service == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Service not found")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, service)
}

// CreateServiceRequest represents create request payload
type CreateServiceRequest struct {
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	Protocol       string  `json:"protocol"`
	Host           string  `json:"host"`
	Port           int     `json:"port"`
	Path           string  `json:"path"`
	ConnectTimeout int     `json:"connect_timeout"`
	ReadTimeout    int     `json:"read_timeout"`
	WriteTimeout   int     `json:"write_timeout"`
	Retries        int     `json:"retries"`
	Enabled        bool    `json:"enabled"`
	UpstreamID     *string `json:"upstream_id"`
	ConnectionID   *string `json:"connection_id"`
}

// Create creates a new logical Gateway service
func (h *ServicesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Service name is required")
		return
	}

	if req.Host != "" {
		if err := egress.ValidateSafeHost(req.Host, serviceEgressPolicy()); err != nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid service host: "+err.Error())
			return
		}
	}

	if req.Port == 0 {
		if req.Protocol == "https" {
			req.Port = 443
		} else {
			req.Port = 80
		}
	}

	service := &models.Service{
		Name:           req.Name,
		Description:    req.Description,
		Protocol:       req.Protocol,
		Host:           req.Host,
		Port:           req.Port,
		Path:           req.Path,
		ConnectTimeout: req.ConnectTimeout,
		ReadTimeout:    req.ReadTimeout,
		WriteTimeout:   req.WriteTimeout,
		Retries:        req.Retries,
		Enabled:        req.Enabled,
		UpstreamID:     req.UpstreamID,
		ConnectionID:   req.ConnectionID,
	}

	if err := h.registry.CreateService(service); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create service: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, service)
}

// Update updates an existing service
func (h *ServicesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	existing, err := h.registry.GetServiceByID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to get service: "+err.Error())
		return
	}
	if existing == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "Service not found")
		return
	}

	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Host != "" {
		if err := egress.ValidateSafeHost(req.Host, serviceEgressPolicy()); err != nil {
			httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid service host: "+err.Error())
			return
		}
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Description = req.Description
	if req.Protocol != "" {
		existing.Protocol = req.Protocol
	}
	existing.Host = req.Host
	if req.Port > 0 {
		existing.Port = req.Port
	}
	existing.Path = req.Path
	if req.ConnectTimeout > 0 {
		existing.ConnectTimeout = req.ConnectTimeout
	}
	if req.ReadTimeout > 0 {
		existing.ReadTimeout = req.ReadTimeout
	}
	if req.WriteTimeout > 0 {
		existing.WriteTimeout = req.WriteTimeout
	}
	existing.Retries = req.Retries
	existing.Enabled = req.Enabled
	existing.UpstreamID = req.UpstreamID
	existing.ConnectionID = req.ConnectionID

	svc := existing.ToService()
	if err := h.registry.UpdateService(svc); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update service: "+err.Error())
		return
	}

	httputil.WriteJSON(w, http.StatusOK, existing)
}

// Delete deletes a service
func (h *ServicesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := h.registry.DeleteService(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete service: "+err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Service deleted successfully"})
}

// Routes returns all routes attached to this service
func (h *ServicesHandler) Routes(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	routes, err := h.registry.ListEndpointsByServiceID(id)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list routes: "+err.Error())
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, routes)
}
