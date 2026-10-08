package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rs/cors"
	"github.com/apirout/apirout/internal/config"
	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/executor"
	"github.com/apirout/apirout/internal/console/middleware"
	"github.com/apirout/apirout/internal/console/monitoring"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/console/server"
)

const shutdownGracePeriod = 15 * time.Second

// App holds the Control Plane application state
type App struct {
	cfg          *config.Config
	logger       *slog.Logger
	reg          *registry.Registry
	pub          *publisher.RedisPublisher
	adminHandler http.Handler
	guiHandler   http.Handler
	closers      []func()
}

// New builds the Control Plane application
func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	reg, err := registry.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("init registry: %w", err)
	}

	if err := reg.Migrate(); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if err := reg.MigrateSchema(); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	logger.Info("Control Plane registry ready", "db_type", reg.DBType())

	// Redis Publisher
	pub := publisher.NewRedisPublisher(cfg.RedisURL, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword, logger)
	reg.SetPublisher(pub)

	authService := auth.NewService(cfg.JWTSecret, reg, int(cfg.JWTTTL.Minutes()), cfg.AdminSecret)
	executorService := executor.NewService(reg, cfg.EncryptionKey)
	monitoringService := monitoring.NewService(reg)
	openapiService := openapi.NewService(reg, executorService)
	postmanService := postman.NewService(reg, executorService)

	loginLimiter := middleware.NewRateLimiter(5, time.Minute, true)

	deps := routerDeps{
		cfg:               cfg,
		reg:               reg,
		authService:       authService,
		executorService:   executorService,
		monitoringService: monitoringService,
		openapiService:    openapiService,
		postmanService:    postmanService,
		publisher:         pub,
		loginLimiter:      loginLimiter,
	}

	adminRouter := buildAdminAPIRouter(deps)
	guiRouter := buildGUIRouter(deps, adminRouter)

	corsOrigins := cfg.CORSAllowedOrigins
	allowCredentials := true
	for _, origin := range corsOrigins {
		if origin == "*" {
			allowCredentials = false
			break
		}
	}
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
		allowCredentials = false
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With", "X-API-Key", "X-Admin-Secret", "X-Apirout-Admin-Secret"},
		AllowCredentials: allowCredentials,
	})

	wrapMiddleware := func(h http.Handler) http.Handler {
		return middleware.RequestID(
			middleware.AccessLog(logger)(
				middleware.Recoverer(logger)(
					middleware.RequestTimer(
						middleware.SecurityHeaders(corsHandler.Handler(h))))))
	}

	adminHandler := wrapMiddleware(adminRouter)
	guiHandler := wrapMiddleware(guiRouter)

	stopHealthChecker := server.StartBackgroundHealthChecker(reg, pub, logger)

	return &App{
		cfg:          cfg,
		logger:       logger,
		reg:          reg,
		pub:          pub,
		adminHandler: adminHandler,
		guiHandler:   guiHandler,
		closers: []func(){
			loginLimiter.Stop,
			stopHealthChecker,
		},
	}, nil
}

// Run starts the dedicated HTTP servers for Admin API (:8001) and Web GUI (:8002)
func (a *App) Run(ctx context.Context) error {
	// If ports are identical, run single combined server
	if a.cfg.AdminPort == a.cfg.GUIPort {
		srv := &http.Server{
			Addr:              ":" + a.cfg.AdminPort,
			Handler:           a.guiHandler,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			a.logger.Info("Combined Control Plane listening", "addr", srv.Addr, "env", a.cfg.AppEnv)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		}
	}

	// Kong-style Dedicated Multi-Port Listeners
	adminSrv := &http.Server{
		Addr:              ":" + a.cfg.AdminPort,
		Handler:           a.adminHandler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	guiSrv := &http.Server{
		Addr:              ":" + a.cfg.GUIPort,
		Handler:           a.guiHandler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		a.logger.Info("Admin Management REST API listening", "addr", adminSrv.Addr, "port", a.cfg.AdminPort)
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("admin api server: %w", err)
		}
	}()

	go func() {
		a.logger.Info("Web Console GUI listening", "addr", guiSrv.Addr, "port", a.cfg.GUIPort)
		if err := guiSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("gui console server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.logger.Info("Control plane shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()
		var sErr []error
		if err := adminSrv.Shutdown(shutdownCtx); err != nil {
			sErr = append(sErr, err)
		}
		if err := guiSrv.Shutdown(shutdownCtx); err != nil {
			sErr = append(sErr, err)
		}
		return errors.Join(sErr...)
	}
}

// Close releases resources
func (a *App) Close() error {
	var errs []error
	for _, closer := range a.closers {
		closer()
	}
	if a.pub != nil {
		_ = a.pub.Close()
	}
	if a.reg != nil {
		if err := a.reg.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
