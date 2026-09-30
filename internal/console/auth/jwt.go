package auth

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/console/registry"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

// Claims represents JWT claims for admin users
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"typ"`
	jwt.RegisteredClaims
}

// Service handles authentication
type Service struct {
	jwtSecret   []byte
	registry    *registry.Registry
	ttl         time.Duration
	adminSecret string
}

// NewService creates a new auth service
func NewService(jwtSecret string, reg *registry.Registry, ttlMinutes int, adminSecret ...string) *Service {
	sec := ""
	if len(adminSecret) > 0 {
		sec = adminSecret[0]
	}
	return &Service{
		jwtSecret:   []byte(jwtSecret),
		registry:    reg,
		ttl:         time.Duration(ttlMinutes) * time.Minute,
		adminSecret: sec,
	}
}

// GenerateToken creates a new JWT for an admin user
func (s *Service) GenerateToken(user *registry.User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.ttl)

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Type:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	return tokenString, expiresAt, err
}

// ValidateToken validates a JWT and returns claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// AuthMiddleware is a mux-compatible middleware for subrouter.Use()
func (s *Service) AuthMiddleware(next http.Handler) http.Handler {
	return s.RequireAuth(next)
}

// RequireAuth middleware that validates JWT or Admin Secret
func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check Hasura-style admin secret header for automation / CI/CD
		if s.adminSecret != "" {
			secretHeader := r.Header.Get("X-Apirout-Admin-Secret")
			if secretHeader == "" {
				secretHeader = r.Header.Get("X-Hasura-Admin-Secret")
			}
			if secretHeader == "" {
				secretHeader = r.Header.Get("X-Admin-Secret")
			}
			if secretHeader != "" && subtle.ConstantTimeCompare([]byte(secretHeader), []byte(s.adminSecret)) == 1 {
				adminUserID := "00000000-0000-0000-0000-000000000001"
				if s.registry != nil {
					if u, _ := s.registry.GetUserByEmail("admin@apirout.io"); u != nil {
						adminUserID = u.ID
					}
				}
				adminClaims := &Claims{
					UserID: adminUserID,
					Email:  "admin@apirout.io",
					Role:   "SUPER_ADMIN",
					Type:   "admin",
				}
				ctx := context.WithValue(r.Context(), UserContextKey, adminClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"Invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		claims, err := s.ValidateToken(parts[1])
		if err != nil {
			http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Only allow admin tokens
		if claims.Type != "admin" {
			http.Error(w, `{"error":"Invalid token type"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware that checks user role
func (s *Service) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*Claims)
			if !ok {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			allowed := false
			for _, role := range roles {
				if claims.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext extracts user claims from context
func GetUserFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(UserContextKey).(*Claims)
	return claims
}
