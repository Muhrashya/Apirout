package config

import (
	"os"
	"strings"
	"testing"
)

func resetEnv() {
	vars := []string{
		"ROLE", "APP_ENV", "PROXY_PORT", "ADMIN_PORT", "GUI_PORT", "STATUS_PORT",
		"ADMIN_ALLOWED_IPS", "ADMIN_SECRET", "JWT_SECRET", "APP_ENCRYPTION_KEY",
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_SSLMODE",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "ADMIN_EMAIL", "ADMIN_PASSWORD",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func TestConfig_DevelopmentDefaults(t *testing.T) {
	resetEnv()
	os.Setenv("APP_ENV", "development")
	defer resetEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed in development mode, got: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv to be development, got: %s", cfg.AppEnv)
	}
	if cfg.ProxyPort != "8000" {
		t.Errorf("expected default ProxyPort 8000, got: %s", cfg.ProxyPort)
	}
}

func TestConfig_Production_AdminSecretValidation(t *testing.T) {
	resetEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "very-strong-production-jwt-secret-key-32-chars!")
	os.Setenv("APP_ENCRYPTION_KEY", "very-strong-production-aes-key-32-chars-long!")
	os.Setenv("DB_PASSWORD", "custom-secure-database-password-2026!")
	defer resetEnv()

	// 1. Missing ADMIN_SECRET
	os.Unsetenv("ADMIN_SECRET")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_SECRET") {
		t.Fatalf("expected error regarding missing ADMIN_SECRET, got: %v", err)
	}

	// 2. Short ADMIN_SECRET (< 24 chars)
	os.Setenv("ADMIN_SECRET", "too-short-secret")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "at least 24 characters") {
		t.Fatalf("expected error regarding short ADMIN_SECRET, got: %v", err)
	}

	// 3. Default placeholder ADMIN_SECRET
	os.Setenv("ADMIN_SECRET", "MasterSecretKey2026!")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "must not use default placeholder") {
		t.Fatalf("expected error regarding default ADMIN_SECRET, got: %v", err)
	}
}

func TestConfig_Production_DBPasswordValidation(t *testing.T) {
	resetEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "very-strong-production-jwt-secret-key-32-chars!")
	os.Setenv("APP_ENCRYPTION_KEY", "very-strong-production-aes-key-32-chars-long!")
	os.Setenv("ADMIN_SECRET", "very-strong-production-admin-secret-24-chars!")
	defer resetEnv()

	// 1. Default DB_PASSWORD
	os.Setenv("DB_PASSWORD", "Password123!")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected error regarding default DB_PASSWORD, got: %v", err)
	}

	// 2. Production default sample
	os.Setenv("DB_PASSWORD", "ProductionSecurePassword123!")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected error regarding default DB_PASSWORD, got: %v", err)
	}

	// 3. Empty DB_PASSWORD for postgres
	os.Setenv("DB_PASSWORD", "")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "DB_PASSWORD") {
		t.Fatalf("expected error regarding empty DB_PASSWORD, got: %v", err)
	}
}

func TestConfig_Production_ValidSecrets(t *testing.T) {
	resetEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "very-strong-production-jwt-secret-key-32-chars!")
	os.Setenv("APP_ENCRYPTION_KEY", "very-strong-production-aes-key-32-chars-long!")
	os.Setenv("ADMIN_SECRET", "very-strong-production-admin-secret-24-chars!")
	os.Setenv("DB_PASSWORD", "genuine-unique-production-db-password-2026!")
	defer resetEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed with valid production secrets, got: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("expected AppEnv production, got: %s", cfg.AppEnv)
	}
	if cfg.AdminSecret != "very-strong-production-admin-secret-24-chars!" {
		t.Errorf("expected AdminSecret to match, got: %s", cfg.AdminSecret)
	}
}
