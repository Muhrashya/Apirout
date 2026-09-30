package registry

import (
	"database/sql"
	"time"
)

// SystemSetting represents a persistent key-value configuration
type SystemSetting struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// GetSetting retrieves a setting value by key
func (r *Registry) GetSetting(key string) (string, error) {
	var val string
	query := "SELECT value FROM system_settings WHERE key = ?"
	err := r.db.Get(&val, r.db.Rebind(query), key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetSetting upserts a setting key-value pair
func (r *Registry) SetSetting(key, value string) error {
	now := time.Now()
	if r.dbType == "postgres" || r.dbType == "postgresql" {
		query := `INSERT INTO system_settings (key, value, updated_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at`
		_, err := r.db.Exec(query, key, value, now)
		return err
	} else if r.dbType == "mysql" {
		query := `INSERT INTO system_settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = VALUES(updated_at)`
		_, err := r.db.Exec(query, key, value, now)
		return err
	} else {
		query := `INSERT INTO system_settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`
		_, err := r.db.Exec(query, key, value, now)
		return err
	}
}

// GetBrandingSettings returns all branding configuration with sane defaults
func (r *Registry) GetBrandingSettings() map[string]string {
	defaults := map[string]string{
		"app_name":      "Apirout",
		"tagline":       "API GATEWAY",
		"logo_url":      "/logo.png",
		"favicon_url":   "/favicon.png",
		"browser_title": "Apirout Console",
	}

	for k, defaultVal := range defaults {
		if val, err := r.GetSetting("branding:" + k); err == nil && val != "" {
			defaults[k] = val
		} else {
			defaults[k] = defaultVal
		}
	}
	return defaults
}

// SaveBrandingSettings saves all branding fields to persistent settings
func (r *Registry) SaveBrandingSettings(settings map[string]string) error {
	for k, v := range settings {
		if err := r.SetSetting("branding:"+k, v); err != nil {
			return err
		}
	}
	return nil
}
