package apiclientauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/core/httputil"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles external consumer login requests
type Handler struct {
	service *Service
}

// NewHandler creates a new API client auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// LoginRequest payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse envelope
type LoginResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Token   string `json:"token"`
		Expired string `json:"expired"`
	} `json:"data"`
}

// Login handles POST /api/login for external machine/API consumers
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	client, err := h.service.GetReader().GetAPIClientByUsername(req.Username)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if client == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(client.PasswordHash), []byte(req.Password)); err != nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, expiresAt, err := h.service.GenerateToken(client)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	var resp LoginResponse
	resp.Status = true
	resp.Message = "Login successful"
	resp.Data.Token = token
	resp.Data.Expired = expiresAt.Format(time.RFC3339)

	httputil.WriteJSON(w, http.StatusOK, resp)
}
