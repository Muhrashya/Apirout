package server

import (
	"encoding/json"
	"net/http"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"
)

// BrandingHandler handles branding & white-label settings
type BrandingHandler struct {
	registry *registry.Registry
}

// NewBrandingHandler creates a new branding handler
func NewBrandingHandler(reg *registry.Registry) *BrandingHandler {
	return &BrandingHandler{registry: reg}
}

// Get returns public branding settings
func (h *BrandingHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings := h.registry.GetBrandingSettings()
	httputil.WriteJSON(w, http.StatusOK, settings)
}

// Update saves branding settings
func (h *BrandingHandler) Update(w http.ResponseWriter, r *http.Request) {
	var input map[string]string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.registry.SaveBrandingSettings(input); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to save branding: "+err.Error())
		return
	}

	settings := h.registry.GetBrandingSettings()
	httputil.WriteJSON(w, http.StatusOK, settings)
}
