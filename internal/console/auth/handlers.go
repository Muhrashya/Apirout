package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/registry"

	"golang.org/x/crypto/bcrypt"
)

// Handler handles auth HTTP requests
type Handler struct {
	service  *Service
	registry *registry.Registry
}

// NewHandler creates a new auth handler
func NewHandler(service *Service, reg *registry.Registry) *Handler {
	return &Handler{
		service:  service,
		registry: reg,
	}
}

// LoginRequest represents login request body (Admin uses EMAIL only)
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	User      struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

// Email validation regex (basic)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Login handles admin login (EMAIL + PASSWORD only)
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	clientIP := httputil.GetClientIP(r)

	// Strict: email is required (not username)
	if req.Email == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "email is required")
		return
	}

	if req.Password == "" {
		httputil.WriteJSONError(w, http.StatusBadRequest, "password is required")
		return
	}

	// Validate email format
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRegex.MatchString(req.Email) {
		httputil.WriteJSONError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	user, err := h.registry.GetUserByEmail(req.Email)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if user == nil {
		// Log failed attempt
		h.registry.CreateAuditLog(&registry.AuditLog{
			EventType: "admin_login_failed",
			IPAddress: &clientIP,
			Details:   `{"email":"` + req.Email + `","reason":"user not found"}`,
			Success:   false,
		})
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Log failed attempt
		h.registry.CreateAuditLog(&registry.AuditLog{
			EventType: "admin_login_failed",
			UserID:    &user.ID,
			IPAddress: &clientIP,
			Details:   `{"email":"` + req.Email + `","reason":"invalid password"}`,
			Success:   false,
		})
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, expiresAt, err := h.service.GenerateToken(user)
	if err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Log successful login
	h.registry.CreateAuditLog(&registry.AuditLog{
		EventType: "admin_login_success",
		UserID:    &user.ID,
		IPAddress: &clientIP,
		Success:   true,
	})

	resp := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	resp.User.ID = user.ID
	resp.User.Email = user.Email
	resp.User.Role = user.Role

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// GroupMembership represents user's membership in a group with permissions
type GroupMembership struct {
	GroupID              string `json:"group_id"`
	GroupName            string `json:"group_name"`
	CanManageConnections bool   `json:"can_manage_connections"`
	CanManageQueries     bool   `json:"can_manage_queries"`
	CanManageEndpoints   bool   `json:"can_manage_endpoints"`
	CanExecuteEndpoints  bool   `json:"can_execute_endpoints"`
	CanViewDocs          bool   `json:"can_view_docs"`
	CanViewDashboard     bool   `json:"can_view_dashboard"`
	CanViewAuditLogs     bool   `json:"can_view_audit_logs"`
}

// MeResponse represents /auth/me response with group permissions
type MeResponse struct {
	ID     string            `json:"id"`
	Email  string            `json:"email"`
	Role   string            `json:"role"`
	Groups []GroupMembership `json:"groups"`
}

// Me returns current user info with group permissions
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.registry.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "User not found")
		return
	}

	// Get user's group memberships with permissions
	userGroups, err := h.registry.GetUserGroups(user.ID)
	if err != nil {
		log.Printf("ERROR: Failed to get user groups for %s: %v", user.ID, err)
		userGroups = []registry.UserGroup{}
	}
	// Debug log
	log.Printf("Me endpoint: User %s (%s) has %d groups", user.ID, user.Email, len(userGroups))
	for _, ug := range userGroups {
		log.Printf("  - Group %s: ManageEndpoints=%v, ManageConn=%v", ug.GroupID, ug.CanManageEndpoints, ug.CanManageConnections)
	}

	// Build group memberships with names
	groups := make([]GroupMembership, 0, len(userGroups))
	for _, ug := range userGroups {
		group, err := h.registry.GetGroupByID(ug.GroupID)
		groupName := ""
		if err == nil && group != nil {
			groupName = group.Name
		}

		groups = append(groups, GroupMembership{
			GroupID:              ug.GroupID,
			GroupName:            groupName,
			CanManageConnections: ug.CanManageConnections,
			CanManageQueries:     ug.CanManageQueries,
			CanManageEndpoints:   ug.CanManageEndpoints,
			CanExecuteEndpoints:  ug.CanExecuteEndpoints,
			CanViewDocs:          ug.CanViewDocs,
			CanViewDashboard:     ug.CanViewDashboard,
			CanViewAuditLogs:     ug.CanViewAuditLogs,
		})
	}

	httputil.WriteJSON(w, http.StatusOK, MeResponse{
		ID:     user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Groups: groups,
	})
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword allows the authenticated user to change their password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.NewPassword) < 8 {
		httputil.WriteJSONError(w, http.StatusBadRequest, "New password must be at least 8 characters")
		return
	}

	user, err := h.registry.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		httputil.WriteJSONError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		httputil.WriteJSONError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	if err := h.registry.UpdateUserPassword(user.ID, req.NewPassword); err != nil {
		httputil.WriteJSONError(w, http.StatusInternalServerError, "Failed to update password: "+err.Error())
		return
	}

	clientIP := httputil.GetClientIP(r)
	h.registry.CreateAuditLog(&registry.AuditLog{
		EventType: "user_password_changed",
		UserID:    &user.ID,
		IPAddress: &clientIP,
		Success:   true,
	})

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

