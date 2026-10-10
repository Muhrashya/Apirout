package server

import (
	"net/http"
	"strings"

	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/core/httputil"
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
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Authentication required to access API documentation")
		return
	}

	var groupIDs []string
	if claims.Role != "SUPER_ADMIN" {
		if gIDs, err := h.registry.GetUserGroupIDs(claims.UserID); err == nil {
			groupIDs = gIDs
		} else {
			groupIDs = []string{}
		}
	}
	
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

	spec := h.openapiService.GenerateSpec(groupIDs, endpointIDs)
	httputil.WriteJSON(w, http.StatusOK, spec)
}

// Postman returns Postman collection
func (h *DocsHandler) Postman(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Authentication required to access API documentation")
		return
	}

	baseURL := r.URL.Query().Get("base_url")
	if baseURL == "" {
		baseURL = "https://api.apirout.io"
	}

	var groupIDs []string
	if claims.Role != "SUPER_ADMIN" {
		if gIDs, err := h.registry.GetUserGroupIDs(claims.UserID); err == nil {
			groupIDs = gIDs
		} else {
			groupIDs = []string{}
		}
	}

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
	
	collection := h.postmanService.GenerateCollection(groupIDs, endpointIDs, baseURL)
	httputil.WriteJSON(w, http.StatusOK, collection)
}

