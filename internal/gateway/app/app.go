package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/apirout/apirout/internal/config"
	"github.com/apirout/apirout/internal/core/cache"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/executor"
	"github.com/apirout/apirout/internal/gateway/middleware"
	"github.com/apirout/apirout/internal/gateway/registry"
	"github.com/apirout/apirout/internal/gateway/responder"
)

const shutdownGracePeriod = 15 * time.Second

// App represents the Data Plane Gateway application
type App struct {
	cfg           *config.Config
	logger        *slog.Logger
	reader        *registry.Reader
	redisSub      *cache.RedisSubscriber
	closers       []func()
	proxyHandler  http.Handler
	statusHandler http.Handler
}

// New creates and initializes the Data Plane Gateway application
func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	reader, err := registry.NewReader(cfg.DBType, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		return nil, fmt.Errorf("init registry reader: %w", err)
	}

	app := &App{
		cfg:    cfg,
		logger: logger,
		reader: reader,
	}
	app.closers = append(app.closers, func() { _ = reader.Close() })

	endpointCache := cache.NewEndpointCache(10 * time.Minute)
	endpointCache.StartCleanupRoutine(2 * time.Minute)
	app.closers = append(app.closers, func() { endpointCache.Stop() })

	connPool := cache.NewManagedConnectionPool(15*time.Minute, 50)
	app.closers = append(app.closers, func() { connPool.Close() })

	countCache := cache.NewCountCache(60 * time.Second)
	countCache.StartCleanupRoutine(5 * time.Minute)
	app.closers = append(app.closers, func() { countCache.Stop() })

	app.redisSub = cache.NewRedisSubscriber(
		cfg.RedisURL, cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword,
		endpointCache, connPool, countCache, logger,
	)

	apiClientAuthService := apiclientauth.NewService(cfg.JWTSecret, reader, cfg.APITokenTTL)
	executorService := executor.NewService(reader, cfg.EncryptionKey, endpointCache, connPool, countCache)

	loginLimiter := middleware.NewRateLimiter(10, time.Minute, true)
	app.closers = append(app.closers, loginLimiter.Stop)

	apiClientHandler := apiclientauth.NewHandler(apiClientAuthService)
	authEngine := auth.NewEngine(reader, cfg.JWTSecret)
	outboundInjector := auth.NewOutboundInjector(cfg.EncryptionKey)
	instantHandler := executor.NewInstantCRUDHandler(reader, cfg.EncryptionKey, connPool, apiClientAuthService, cfg.AdminSecret, cfg.JWTSecret)
	executorHandler := executor.NewHandler(executorService, reader, apiClientAuthService, authEngine, outboundInjector, instantHandler)

	// Wire zero-downtime atomic pointer swapping on EventBus route invalidation
	app.redisSub.SetOnRouteInvalidated(func() {
		if err := executorHandler.ReloadRouterFromDB(); err != nil {
			logger.Error("Failed to reload in-memory Radix router upon invalidation", "error", err)
		} else {
			logger.Info("In-memory Radix router dynamically swapped upon invalidation")
		}
	})

	// Wire Control Plane target health transitions into edge circuit breaker
	app.redisSub.SetOnTargetHealthChanged(func(targetID, action string) {
		executorHandler.OnTargetHealthChanged(targetID, action)
		logger.Info("Target circuit breaker state updated via Control Plane health event", "target_id", targetID, "action", action)
	})

	// Public Consumer Gateway Router (Port 8000)
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		responder.JSON(w, r, http.StatusOK, map[string]string{
			"status": "ok",
			"plane":  "data-plane",
			"engine": "apirout",
		})
	}).Methods("GET")

	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Hasura-style Instant Database REST API (protected with auth)
	instantRouter := r.PathPrefix("/api/v1/instant/{connId}").Subrouter()
	instantRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := instantHandler.AuthenticateRequest(r); err != nil {
				responder.Error(w, r, http.StatusUnauthorized, responder.ErrCodeUnauthorized, err.Error())
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	instantRouter.HandleFunc("/schema", instantHandler.GetSchema).Methods("GET")
	instantRouter.HandleFunc("/{table}", instantHandler.QueryTable).Methods("GET")
	instantRouter.HandleFunc("/{table}", instantHandler.InsertRow).Methods("POST")
	instantRouter.HandleFunc("/{table}/{id}", instantHandler.GetRow).Methods("GET")
	instantRouter.HandleFunc("/{table}/{id}", instantHandler.UpdateRow).Methods("PUT", "PATCH")
	instantRouter.HandleFunc("/{table}/{id}", instantHandler.DeleteRow).Methods("DELETE")

	r.Handle("/api/login", loginLimiter.Handler(http.HandlerFunc(apiClientHandler.Login))).Methods("POST")
	r.PathPrefix("/api/").HandlerFunc(executorHandler.Execute)

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

	corsOptions := cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Requested-With", "X-API-Key", "X-Admin-Secret", "X-Apirout-Admin-Secret", "X-Request-ID", "X-Correlation-ID"},
		AllowCredentials: allowCredentials,
	}
	corsMiddleware := cors.New(corsOptions)
	app.proxyHandler = responder.Middleware(middleware.MetricsTracker(corsMiddleware.Handler(r)))

	// Status & Metrics Router (Port 8100)
	statusR := mux.NewRouter()
	statusR.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"plane":   "data-plane",
			"service": "apirout-status",
		})
	}).Methods("GET")
	statusR.Handle("/metrics", promhttp.Handler()).Methods("GET")
	app.statusHandler = statusR

	return app, nil
}

// Run starts the Data Plane Gateway HTTP server and background listeners
func (a *App) Run(ctx context.Context) error {
	a.redisSub.Start(ctx)
	defer a.redisSub.Stop()

	// If StatusPort equals ProxyPort, run single listener
	if a.cfg.StatusPort == a.cfg.ProxyPort {
		server := &http.Server{
			Addr:         ":" + a.cfg.ProxyPort,
			Handler:      a.proxyHandler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		serverErr := make(chan error, 1)
		go func() {
			a.logger.Info("Data Plane Gateway listening", "port", a.cfg.ProxyPort)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErr <- err
			}
		}()

		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
			defer cancel()
			return server.Shutdown(shutdownCtx)
		case err := <-serverErr:
			return fmt.Errorf("gateway server error: %w", err)
		}
	}

	// Kong-style Dedicated Multi-Port Listeners:
	// ProxyPort (:8000) for API Traffic, StatusPort (:8100) for Prometheus & Health Probes
	proxyServer := &http.Server{
		Addr:         ":" + a.cfg.ProxyPort,
		Handler:      a.proxyHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	statusServer := &http.Server{
		Addr:         ":" + a.cfg.StatusPort,
		Handler:      a.statusHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 2)
	go func() {
		a.logger.Info("Data Plane Proxy listening", "addr", proxyServer.Addr, "port", a.cfg.ProxyPort)
		if err := proxyServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("proxy server: %w", err)
		}
	}()

	go func() {
		a.logger.Info("Data Plane Status & Metrics listening", "addr", statusServer.Addr, "port", a.cfg.StatusPort)
		if err := statusServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("status server: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("Data Plane Gateway shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
		defer cancel()
		var sErr []error
		if err := proxyServer.Shutdown(shutdownCtx); err != nil {
			sErr = append(sErr, err)
		}
		if err := statusServer.Shutdown(shutdownCtx); err != nil {
			sErr = append(sErr, err)
		}
		return errors.Join(sErr...)
	case err := <-serverErr:
		return fmt.Errorf("gateway server error: %w", err)
	}
}

// Close cleans up resources
func (a *App) Close() {
	for i := len(a.closers) - 1; i >= 0; i-- {
		a.closers[i]()
	}
}
