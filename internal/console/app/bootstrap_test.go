package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/config"
)

func TestApp_HeadlessBootstrap(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "bootstrap_test.db")

	cfg := &config.Config{
		Role:                   config.RoleControlPlane,
		AppEnv:                 "development",
		DataDir:                tempDir,
		DBType:                 "sqlite",
		DBPath:                 dbPath,
		BootstrapAdminEmail:    "root@cloud.internal",
		BootstrapAdminPassword: "SuperSecurePass2026!",
		ConsoleJWTSecret:       "console-jwt-secret-at-least-32chars!",
		GatewayJWTSecret:       "gateway-jwt-secret-at-least-32chars!",
		AdminSecret:            "admin-secret-24-chars-long!",
		EncryptionKey:          "aes-encryption-key-32-chars-ok!",
		JWTTTL:                 60 * time.Minute,
		APITokenTTL:            60 * time.Minute,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	app, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("expected New() to succeed, got: %v", err)
	}
	defer func() {
		for _, c := range app.closers {
			c()
		}
		_ = app.reg.Close()
	}()

	hasAdmin, err := app.reg.HasSuperAdmin()
	if err != nil {
		t.Fatalf("failed to check HasSuperAdmin: %v", err)
	}
	if !hasAdmin {
		t.Fatalf("expected HasSuperAdmin to be true after headless bootstrap")
	}

	user, err := app.reg.GetUserByEmail("root@cloud.internal")
	if err != nil || user == nil {
		t.Fatalf("expected bootstrapped super admin user to exist: %v", err)
	}
	if user.Role != "SUPER_ADMIN" {
		t.Errorf("expected role SUPER_ADMIN, got: %s", user.Role)
	}
}
