package registry

import (
	"fmt"
	"time"

	"github.com/apirout/apirout/internal/core/models"
)

// ListGatewayDomains returns all configured gateway domains
func (r *Registry) ListGatewayDomains() ([]models.GatewayDomain, error) {
	var domains []models.GatewayDomain
	err := r.db.Select(&domains, "SELECT id, domain_name, display_name, is_default, environment, COALESCE(default_allowed_ips, '*') as default_allowed_ips, description, created_at, updated_at FROM gateway_domains ORDER BY is_default DESC, domain_name ASC")
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		// Provide default domain if empty
		defaultDomain := models.GatewayDomain{
			ID:                "default-domain-01",
			DomainName:        "localhost:8080",
			DisplayName:       "Default Gateway",
			IsDefault:         true,
			Environment:       "development",
			DefaultAllowedIPs: "*",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		_ = r.CreateGatewayDomain(&defaultDomain)
		domains = append(domains, defaultDomain)
	}
	return domains, nil
}

// CreateGatewayDomain registers a new gateway domain
func (r *Registry) CreateGatewayDomain(domain *models.GatewayDomain) error {
	if domain.ID == "" {
		domain.ID = fmt.Sprintf("dom-%d", time.Now().UnixNano())
	}
	now := time.Now()
	domain.CreatedAt = now
	domain.UpdatedAt = now
	if domain.DefaultAllowedIPs == "" {
		domain.DefaultAllowedIPs = "*"
	}

	if domain.IsDefault {
		// Unset any previous default
		_, _ = r.db.Exec(r.db.Rebind("UPDATE gateway_domains SET is_default = FALSE"))
	}

	query := `INSERT INTO gateway_domains (id, domain_name, display_name, is_default, environment, default_allowed_ips, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(r.db.Rebind(query),
		domain.ID, domain.DomainName, domain.DisplayName, domain.IsDefault, domain.Environment, domain.DefaultAllowedIPs, domain.Description, domain.CreatedAt, domain.UpdatedAt,
	)
	return err
}

// DeleteGatewayDomain removes a gateway domain
func (r *Registry) DeleteGatewayDomain(id string) error {
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM gateway_domains WHERE id = ?"), id)
	return err
}
