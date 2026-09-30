package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/apirout/apirout/internal/config"
	consoleApp "github.com/apirout/apirout/internal/console/app"
	"github.com/apirout/apirout/internal/console/registry"
	gatewayApp "github.com/apirout/apirout/internal/gateway/app"
)

const version = "1.0.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "apirout fatal error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			return runMigrate()
		case "create-admin":
			return runCreateAdmin()
		case "version", "-v", "--version":
			fmt.Printf("Apirout version %s (Open Source Instant API Engine & Gateway)\n", version)
			return nil
		case "help", "-h", "--help":
			printHelp()
			return nil
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg)

	fmt.Printf(`
   _____        .__                     __   
  /  _  \______ |__|______  ____  __ __/  |_ 
 /  /_\  \____ \|  \_  __ \/  _ \|  |  \   __\
/    |    \  |_> >  ||  | \(  <_> )  |  /|  |  
\____|__  /   __/|__||__|   \____/|____/ |__|  
        \/|__|  v%s - Open Source API Engine & Gateway
`, version)

	logger.Info("Starting Apirout Engine", "version", version, "role", cfg.Role, "env", cfg.AppEnv)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cfg.Role {
	case config.RoleTraditional:
		return runAllInOne(ctx, cfg, logger)
	case config.RoleControlPlane:
		return runControlPlane(ctx, cfg, logger)
	case config.RoleDataPlane:
		return runDataPlane(ctx, cfg, logger)
	default:
		return fmt.Errorf("unknown role %q (expected: traditional, control_plane, data_plane)", cfg.Role)
	}
}

func runAllInOne(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("Initializing Apirout in ALL-IN-ONE mode")

	cp, err := consoleApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init control-plane console: %w", err)
	}
	defer cp.Close()

	dp, err := gatewayApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init data-plane gateway: %w", err)
	}
	defer dp.Close()

	logger.Info("Apirout services online (Dedicated Multi-Port Listeners)",
		"gui_url", fmt.Sprintf("http://localhost:%s", cfg.GUIPort),
		"admin_api_url", fmt.Sprintf("http://localhost:%s/api/v1", cfg.AdminPort),
		"proxy_gateway_url", fmt.Sprintf("http://localhost:%s/api/*", cfg.ProxyPort),
		"status_metrics_url", fmt.Sprintf("http://localhost:%s/metrics", cfg.StatusPort),
	)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cp.Run(ctx); err != nil {
			errCh <- fmt.Errorf("control-plane error: %w", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dp.Run(ctx); err != nil {
			errCh <- fmt.Errorf("data-plane error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Apirout received shutdown signal, stopping all services")
	case err := <-errCh:
		logger.Error("Apirout service failed", "error", err)
		return err
	}

	wg.Wait()
	return nil
}

func runControlPlane(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("Initializing Apirout in CONTROL-PLANE mode (Admin API & GUI)")

	cp, err := consoleApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init control-plane: %w", err)
	}
	defer cp.Close()

	logger.Info("Control Plane online",
		"gui_url", fmt.Sprintf("http://localhost:%s", cfg.GUIPort),
		"admin_api_url", fmt.Sprintf("http://localhost:%s/api/v1", cfg.AdminPort),
	)
	return cp.Run(ctx)
}

func runDataPlane(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("Initializing Apirout in DATA-PLANE mode (Stateless Gateway & Status)")

	dp, err := gatewayApp.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init data-plane gateway: %w", err)
	}
	defer dp.Close()

	logger.Info("Data Plane Gateway online",
		"proxy_gateway_url", fmt.Sprintf("http://localhost:%s/api/*", cfg.ProxyPort),
		"status_metrics_url", fmt.Sprintf("http://localhost:%s/metrics", cfg.StatusPort),
	)
	return dp.Run(ctx)
}

func newLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if cfg.LogLevel == "debug" {
		opts.Level = slog.LevelDebug
	}
	if cfg.AppEnv == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func printHelp() {
	fmt.Print(`
Apirout CLI - Open Source Instant API Engine & Gateway (Kong & Hasura Architecture)

Usage:
  apirout              Start the engine (mode controlled by APIROUT_ROLE: traditional, control_plane, data_plane)
  apirout migrate      Run database schema migrations and exit
  apirout create-admin Create super admin user interactively
  apirout version      Show Apirout version
  apirout help         Show this help guide

Operation Modes (Kong Paradigm):
  traditional          Run Control Plane (Admin API + Web GUI) and Data Plane (Gateway + Status) [Default]
  control_plane        Run Admin API (:8001) and Web Console GUI (:8002)
  data_plane           Run API Gateway Proxy (:8000) and Status/Metrics (:8100)

Dedicated Listeners (Kong Architecture):
  APIROUT_PROXY_LISTEN   Public API Gateway Proxy port (default: 8000)
  APIROUT_ADMIN_LISTEN   Admin Management REST API port (default: 8001)
  APIROUT_GUI_LISTEN     Web Console GUI port (default: 8002)
  APIROUT_STATUS_LISTEN  Health & Prometheus Metrics port (default: 8100)
  APIROUT_ADMIN_ALLOWED_IPS Allowed CIDR/IPs for Admin & GUI access (default: * allow all)

Environment Variables (Hasura & Kong Standard):
  APIROUT_ROLE         traditional | control_plane | data_plane (default: traditional)
  APIROUT_DATABASE_URL Full Postgres connection URI
  APIROUT_ADMIN_SECRET Hasura-style master secret for CI/CD and direct API header access
  APIROUT_ADMIN_EMAIL  Initial super admin email (default: admin@apirout.io)
  APIROUT_REDIS_URL    Redis URL for distributed cache & event synchronization
`)
}

func runMigrate() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	reg, err := registry.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		return fmt.Errorf("schema migration failed: %w", err)
	}

	fmt.Println("Apirout database migration completed successfully!")
	return nil
}

func runCreateAdmin() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	reg, err := registry.New(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("connect to db: %w", err)
	}
	defer reg.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Admin Email [admin@apirout.io]: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)
	if email == "" {
		email = "admin@apirout.io"
	}

	fmt.Print("Enter Admin Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if err := reg.EnsureSuperAdmin(email, password); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	fmt.Printf("Admin account %s created successfully!\n", email)
	return nil
}
