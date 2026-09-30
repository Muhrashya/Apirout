package server

import (
	"net/http"

	"strings"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	"github.com/apirout/apirout/internal/console/registry"
)

// DocsHandler handles API documentation
type DocsHandler struct {
	openapiService *openapi.Service
	postmanService *postman.Service
	registry       *registry.Registry
}

// NewDocsHandler creates a new docs handler
func NewDocsHandler(openapiSvc *openapi.Service, postmanSvc *postman.Service, reg *registry.Registry) *DocsHandler {
	return &DocsHandler{
		openapiService: openapiSvc,
		postmanService: postmanSvc,
		registry:       reg,
	}
}

// OpenAPI returns OpenAPI specification
func (h *DocsHandler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	// TODO: Get group IDs from user context for RBAC filtering
	// TODO: Get group IDs from user context for RBAC filtering
	
	var endpointIDs []string
	if ids := r.URL.Query().Get("endpoint_ids"); ids != "" {
		endpointIDs = strings.Split(ids, ",")
	}

	if clientID := r.URL.Query().Get("api_client_id"); clientID != "" {
		allowedIDs, err := h.registry.GetAPIClientAllowedEndpointIDs(clientID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch allowed endpoints for client")
			return
		}
		if allowedIDs == nil {
			endpointIDs = []string{}
		} else {
			endpointIDs = allowedIDs
		}
	}

	spec := h.openapiService.GenerateSpec(nil, endpointIDs)
	httputil.WriteJSON(w, http.StatusOK, spec)
}

// Postman returns Postman collection
func (h *DocsHandler) Postman(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("base_url")
	if baseURL == "" {
		baseURL = "https://api.apirout.io"
	}

	// TODO: Get group IDs from user context for RBAC filtering
	// TODO: Get group IDs from user context for RBAC filtering

	var endpointIDs []string
	if ids := r.URL.Query().Get("endpoint_ids"); ids != "" {
		endpointIDs = strings.Split(ids, ",")
	}

	if clientID := r.URL.Query().Get("api_client_id"); clientID != "" {
		allowedIDs, err := h.registry.GetAPIClientAllowedEndpointIDs(clientID)
		if err != nil {
			httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to fetch allowed endpoints for client")
			return
		}
		if allowedIDs == nil {
			endpointIDs = []string{}
		} else {
			endpointIDs = allowedIDs
		}
	}
	
	collection := h.postmanService.GenerateCollection(nil, endpointIDs, baseURL)
	httputil.WriteJSON(w, http.StatusOK, collection)
}
