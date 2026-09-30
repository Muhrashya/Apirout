package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/models"
)

// DomainHandler handles gateway domains management
type DomainHandler struct {
	registry *registry.Registry
}

// NewDomainHandler creates a new domain handler
func NewDomainHandler(reg *registry.Registry) *DomainHandler {
	return &DomainHandler{registry: reg}
}

// List returns all registered domains
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	domains, err := h.registry.ListGatewayDomains()
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to list gateway domains")
		return
	}
	httputil.WriteJSONData(w, http.StatusOK, domains)
}

// Create registers a new gateway domain
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var domain models.GatewayDomain
	if err := json.NewDecoder(r.Body).Decode(&domain); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	domain.DomainName = strings.ToLower(strings.TrimSpace(domain.DomainName))
	domain.DisplayName = strings.TrimSpace(domain.DisplayName)
	if domain.DomainName == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Domain name is required")
		return
	}
	if domain.DisplayName == "" {
		domain.DisplayName = domain.DomainName
	}
	if domain.Environment == "" {
		domain.Environment = "production"
	}

	if err := h.registry.CreateGatewayDomain(&domain); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to create gateway domain: "+err.Error())
		return
	}

	httputil.WriteJSONData(w, http.StatusCreated, domain)
}

// Delete removes a domain by ID
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Domain ID is required")
		return
	}

	if err := h.registry.DeleteGatewayDomain(id); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete domain")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
