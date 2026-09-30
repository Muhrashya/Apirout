package registry

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"golang.org/x/crypto/bcrypt"
)

// ListConsumers returns all external API consumers with credential counts and route counts
func (r *Registry) ListConsumers() ([]models.ConsumerWithDetails, error) {
	var consumers []models.ConsumerWithDetails
	query := `
		SELECT ac.id, ac.group_id, ac.username, ac.custom_id, ac.info as description,
		       COALESCE(ac.status, 'active') as status, ac.created_at,
		       COALESCE(ac.updated_at, ac.created_at) as updated_at,
		       cg.name as group_name,
		       COALESCE((SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = ac.id), 0) as allowed_endpoints_count,
		       COALESCE((SELECT COUNT(*) FROM consumer_credentials WHERE consumer_id = ac.id), 0) as credentials_count
		FROM api_clients ac
		LEFT JOIN consumer_groups cg ON cg.id = ac.group_id
		ORDER BY ac.username ASC
	`
	err := r.db.Select(&consumers, query)
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

// GetConsumerByID returns a single consumer with detail counts
func (r *Registry) GetConsumerByID(id string) (*models.ConsumerWithDetails, error) {
	var consumer models.ConsumerWithDetails
	query := `
		SELECT ac.id, ac.group_id, ac.username, ac.custom_id, ac.info as description,
		       COALESCE(ac.status, 'active') as status, ac.created_at,
		       COALESCE(ac.updated_at, ac.created_at) as updated_at,
		       cg.name as group_name,
		       COALESCE((SELECT COUNT(*) FROM api_client_endpoints WHERE api_client_id = ac.id), 0) as allowed_endpoints_count,
		       COALESCE((SELECT COUNT(*) FROM consumer_credentials WHERE consumer_id = ac.id), 0) as credentials_count
		FROM api_clients ac
		LEFT JOIN consumer_groups cg ON cg.id = ac.group_id
		WHERE ac.id = ?
	`
	err := r.db.Get(&consumer, r.db.Rebind(query), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &consumer, nil
}

// CreateConsumer creates a new API consumer
func (r *Registry) CreateConsumer(c *models.Consumer, initialPassword string, endpointIDs []string) (*models.Consumer, error) {
	c.ID = generateID()
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	if c.Status == "" {
		c.Status = "active"
	}

	// Generate a fallback password hash if none provided
	if initialPassword == "" {
		initialPassword = generateRandomHex(16)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO api_clients (id, group_id, username, custom_id, info, status, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(r.db.Rebind(query),
		c.ID, c.GroupID, c.Username, c.CustomID, c.Description, c.Status, string(hash), c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Insert endpoint allowlist
	for _, epID := range endpointIDs {
		_, err = tx.Exec(
			r.db.Rebind("INSERT INTO api_client_endpoints (api_client_id, endpoint_id, created_at) VALUES (?, ?, ?)"),
			c.ID, epID, now,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return c, nil
}

// UpdateConsumer updates consumer metadata and status
func (r *Registry) UpdateConsumer(c *models.Consumer, endpointIDs []string) error {
	c.UpdatedAt = time.Now()

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		UPDATE api_clients SET
			group_id = ?, custom_id = ?, info = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.Exec(r.db.Rebind(query),
		c.GroupID, c.CustomID, c.Description, c.Status, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return err
	}

	if endpointIDs != nil {
		_, err = tx.Exec(r.db.Rebind("DELETE FROM api_client_endpoints WHERE api_client_id = ?"), c.ID)
		if err != nil {
			return err
		}

		now := time.Now()
		for _, epID := range endpointIDs {
			_, err = tx.Exec(
				r.db.Rebind("INSERT INTO api_client_endpoints (api_client_id, endpoint_id, created_at) VALUES (?, ?, ?)"),
				c.ID, epID, now,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ==========================================
// CONSUMER GROUPS
// ==========================================

// ListConsumerGroups returns all policy consumer groups
func (r *Registry) ListConsumerGroups() ([]models.ConsumerGroupWithCount, error) {
	var groups []models.ConsumerGroupWithCount
	query := `
		SELECT cg.id, cg.name, cg.description, cg.rate_limit_rpm, cg.rate_limit_burst,
		       cg.created_at, cg.updated_at,
		       COALESCE((SELECT COUNT(*) FROM api_clients ac WHERE ac.group_id = cg.id), 0) as consumers_count
		FROM consumer_groups cg
		ORDER BY cg.name ASC
	`
	err := r.db.Select(&groups, query)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// GetConsumerGroupByID returns a single consumer group by ID
func (r *Registry) GetConsumerGroupByID(id string) (*models.ConsumerGroup, error) {
	var group models.ConsumerGroup
	query := `SELECT id, name, description, rate_limit_rpm, rate_limit_burst, created_at, updated_at FROM consumer_groups WHERE id = ?`
	err := r.db.Get(&group, r.db.Rebind(query), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// CreateConsumerGroup creates a new consumer group
func (r *Registry) CreateConsumerGroup(g *models.ConsumerGroup) error {
	g.ID = generateID()
	now := time.Now()
	g.CreatedAt = now
	g.UpdatedAt = now

	if g.RateLimitRPM <= 0 {
		g.RateLimitRPM = 60
	}
	if g.RateLimitBurst <= 0 {
		g.RateLimitBurst = 10
	}

	query := `
		INSERT INTO consumer_groups (id, name, description, rate_limit_rpm, rate_limit_burst, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(r.db.Rebind(query), g.ID, g.Name, g.Description, g.RateLimitRPM, g.RateLimitBurst, g.CreatedAt, g.UpdatedAt)
	return err
}

// UpdateConsumerGroup updates a consumer group
func (r *Registry) UpdateConsumerGroup(g *models.ConsumerGroup) error {
	g.UpdatedAt = time.Now()

	query := `
		UPDATE consumer_groups SET name = ?, description = ?, rate_limit_rpm = ?, rate_limit_burst = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(r.db.Rebind(query), g.Name, g.Description, g.RateLimitRPM, g.RateLimitBurst, g.UpdatedAt, g.ID)
	return err
}

// DeleteConsumerGroup deletes a consumer group
func (r *Registry) DeleteConsumerGroup(id string) error {
	_, _ = r.db.Exec(r.db.Rebind("UPDATE api_clients SET group_id = NULL WHERE group_id = ?"), id)
	_, err := r.db.Exec(r.db.Rebind("DELETE FROM consumer_groups WHERE id = ?"), id)
	return err
}

// ==========================================
// CONSUMER CREDENTIALS (API Keys lifecycle)
// ==========================================

// ListConsumerCredentials returns all credentials for a consumer (without plaintext secrets)
func (r *Registry) ListConsumerCredentials(consumerID string) ([]models.ConsumerCredential, error) {
	var creds []models.ConsumerCredential
	query := `
		SELECT id, consumer_id, name, key_type, key_id, secret_hash, masked_key, enabled, expires_at, last_used_at, created_at
		FROM consumer_credentials
		WHERE consumer_id = ?
		ORDER BY created_at DESC
	`
	err := r.db.Select(&creds, r.db.Rebind(query), consumerID)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// CreateConsumerCredential generates a new API Key credential for a consumer.
// Returns the secret ONCE in ConsumerCredentialCreated. Plaintext is never stored.
func (r *Registry) CreateConsumerCredential(consumerID, name, keyType string, expiresAt *time.Time) (*models.ConsumerCredentialCreated, error) {
	id := generateID()
	now := time.Now()

	if keyType == "" {
		keyType = "api-key"
	}
	if name == "" {
		name = "Default API Key"
	}

	// Generate key_id (public prefix, e.g. "ak_live_8f3a2b") and raw secret
	prefix := "ak_live_" + generateRandomHex(4)
	secretPart := generateRandomHex(24)
	fullKey := fmt.Sprintf("%s_%s", prefix, secretPart)

	// Hash the full key with bcrypt
	hash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	masked := fmt.Sprintf("%s_...%s", prefix, fullKey[len(fullKey)-4:])

	cred := models.ConsumerCredential{
		ID:         id,
		ConsumerID: consumerID,
		Name:       name,
		KeyType:    keyType,
		KeyID:      prefix,
		SecretHash: string(hash),
		MaskedKey:  masked,
		Enabled:    true,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
	}

	query := `
		INSERT INTO consumer_credentials (
			id, consumer_id, name, key_type, key_id, secret_hash, masked_key, enabled, expires_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.Exec(r.db.Rebind(query),
		cred.ID, cred.ConsumerID, cred.Name, cred.KeyType, cred.KeyID, cred.SecretHash,
		cred.MaskedKey, cred.Enabled, cred.ExpiresAt, cred.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &models.ConsumerCredentialCreated{
		ConsumerCredential: cred,
		KeySecret:          fullKey,
	}, nil
}

// ToggleConsumerCredential enables or disables a credential
func (r *Registry) ToggleConsumerCredential(id string, enabled bool) error {
	query := `UPDATE consumer_credentials SET enabled = ? WHERE id = ?`
	_, err := r.db.Exec(r.db.Rebind(query), enabled, id)
	return err
}

// DeleteConsumerCredential revokes and deletes a credential
func (r *Registry) DeleteConsumerCredential(id string) error {
	query := `DELETE FROM consumer_credentials WHERE id = ?`
	_, err := r.db.Exec(r.db.Rebind(query), id)
	return err
}

// ValidateAPIKey checks if an incoming raw API key matches any active credential
func (r *Registry) ValidateAPIKey(rawKey string) (*models.Consumer, *models.ConsumerCredential, error) {
	if rawKey == "" {
		return nil, nil, fmt.Errorf("empty API key")
	}

	// Extract prefix if key has format "ak_live_xxxx_yyyy"
	parts := strings.Split(rawKey, "_")
	var keyID string
	if len(parts) >= 3 && parts[0] == "ak" {
		keyID = fmt.Sprintf("%s_%s_%s", parts[0], parts[1], parts[2])
	}

	var creds []models.ConsumerCredential
	if keyID != "" {
		query := `SELECT * FROM consumer_credentials WHERE key_id = ? AND enabled = TRUE`
		_ = r.db.Select(&creds, r.db.Rebind(query), keyID)
	}

	// Fallback: check all enabled credentials if prefix search misses
	if len(creds) == 0 {
		query := `SELECT * FROM consumer_credentials WHERE enabled = TRUE`
		_ = r.db.Select(&creds, query)
	}

	for _, cred := range creds {
		// Check expiration
		if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
			continue
		}

		if err := bcrypt.CompareHashAndPassword([]byte(cred.SecretHash), []byte(rawKey)); err == nil {
			// Update last_used_at async
			go func(cID string) {
				_, _ = r.db.Exec("UPDATE consumer_credentials SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1", cID)
			}(cred.ID)

			// Get consumer
			var consumer models.Consumer
			err := r.db.Get(&consumer, r.db.Rebind("SELECT id, group_id, username, custom_id, info as description, status, created_at FROM api_clients WHERE id = ?"), cred.ConsumerID)
			if err != nil {
				return nil, nil, err
			}

			if consumer.Status == "suspended" || consumer.Status == "revoked" {
				return nil, nil, fmt.Errorf("consumer account is %s", consumer.Status)
			}

			return &consumer, &cred, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid API key")
}

func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "abcd1234efgh5678"
	}
	return hex.EncodeToString(bytes)
}
