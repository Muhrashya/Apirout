package apiclientauth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/registry"
)

type contextKey string

const APIClientContextKey contextKey = "api_client"

// Claims represents JWT claims for external API consumers
type Claims struct {
	APIClientID string `json:"api_client_id"`
	Username    string `json:"username"`
	GroupID     string `json:"group_id"`
	Type        string `json:"typ"`
	jwt.RegisteredClaims
}

// Service manages API client authentication
type Service struct {
	jwtSecret []byte
	reader    *registry.Reader
	ttl       time.Duration
}

// NewService creates a new API client auth service
func NewService(jwtSecret string, reader *registry.Reader, ttl time.Duration) *Service {
	return &Service{
		jwtSecret: []byte(jwtSecret),
		reader:    reader,
		ttl:       ttl,
	}
}

// GenerateToken creates a signed JWT for an authenticated API client
func (s *Service) GenerateToken(client *models.APIClient) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.ttl)

	groupID := ""
	if client.GroupID != nil {
		groupID = *client.GroupID
	}

	claims := Claims{
		APIClientID: client.ID,
		Username:    client.Username,
		GroupID:     groupID,
		Type:        "api_client",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	return tokenString, expiresAt, err
}

// ValidateToken parses and verifies an API client JWT token string
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

// GetReader returns the underlying registry reader
func (s *Service) GetReader() *registry.Reader {
	return s.reader
}
