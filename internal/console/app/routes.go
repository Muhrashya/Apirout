package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/apirout/apirout/internal/config"
	"github.com/apirout/apirout/internal/console/auth"
	"github.com/apirout/apirout/internal/console/executor"
	"github.com/apirout/apirout/internal/console/gitops"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/console/middleware"
	"github.com/apirout/apirout/internal/console/monitoring"
	"github.com/apirout/apirout/internal/console/openapi"
	"github.com/apirout/apirout/internal/console/postman"
	"github.com/apirout/apirout/internal/console/publisher"
	"github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/console/server"
	"github.com/apirout/apirout/internal/core/iputil"
)

type routerDeps struct {
	cfg               *config.Config
	reg               *registry.Registry
	authService       *auth.Service
	executorService   *executor.Service
	monitoringService *monitoring.Service
	openapiService    *openapi.Service
	postmanService    *postman.Service
	publisher         *publisher.RedisPublisher
	loginLimiter      *middleware.RateLimiter
}

// buildAdminAPIRouter builds the dedicated Admin Management REST API router (Port 8001)
func buildAdminAPIRouter(d routerDeps) *mux.Router {
	r := mux.NewRouter()

	// Optional IP allowlist middleware for Admin operations
	if d.cfg.AdminAllowedIPs != "" && d.cfg.AdminAllowedIPs != "*" {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				clientIP := iputil.ExtractClientIP(req)
				if !iputil.IsIPAllowed(clientIP, d.cfg.AdminAllowedIPs) {
					httputil.WriteJSONError(w, http.StatusForbidden, "IP not allowed to access Admin API")
					return
				}
				next.ServeHTTP(w, req)
			})
		})
	}

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"plane":   "control-plane",
			"service": "apirout-admin-api",
		})
	}).Methods("GET")

	authHandler := auth.NewHandler(d.authService, d.reg)

	// API V1 Subrouter for Admin Management
	apiV1 := r.PathPrefix("/api/v1").Subrouter()
	apiV1.Handle("/auth/login", d.loginLimiter.Handler(http.HandlerFunc(authHandler.Login))).Methods("POST")

	// Branding & White-Labeling (GET is public, PUT is ADMIN/SUPER_ADMIN only)
	brandingHandler := server.NewBrandingHandler(d.reg)
	apiV1.HandleFunc("/branding", brandingHandler.Get).Methods("GET")

	// Protected Admin Routes
	adminRouter := apiV1.PathPrefix("").Subrouter()
	adminRouter.Use(d.authService.RequireAuth)
	adminRouter.Handle("/branding", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(brandingHandler.Update))).Methods("PUT", "POST")
	adminRouter.Handle("/auth/me", http.HandlerFunc(authHandler.Me)).Methods("GET")
	adminRouter.Handle("/auth/change-password", http.HandlerFunc(authHandler.ChangePassword)).Methods("POST")
	adminRouter.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"plane":   "control-plane",
			"service": "apirout-admin-api",
		})
	}).Methods("GET")

	// Users management (SUPER_ADMIN only)
	userHandler := server.NewUserHandler(d.reg)
	adminRouter.Handle("/users", http.HandlerFunc(userHandler.List)).Methods("GET")
	adminRouter.Handle("/users", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(userHandler.Create))).Methods("POST")
	adminRouter.Handle("/users/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(userHandler.Update))).Methods("PUT")
	adminRouter.Handle("/users/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(userHandler.Delete))).Methods("DELETE")

	// Groups management
	groupHandler := server.NewGroupHandler(d.reg)
	adminRouter.HandleFunc("/groups", groupHandler.List).Methods("GET")
	adminRouter.Handle("/groups", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(groupHandler.Create))).Methods("POST")
	adminRouter.Handle("/groups/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(groupHandler.Update))).Methods("PUT")
	adminRouter.Handle("/groups/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(groupHandler.Delete))).Methods("DELETE")

	// Group members
	adminRouter.Handle("/groups/{id}/members", http.HandlerFunc(groupHandler.ListMembers)).Methods("GET")
	adminRouter.Handle("/groups/{id}/members", http.HandlerFunc(groupHandler.AddMember)).Methods("POST")
	adminRouter.Handle("/groups/{id}/members/{userId}", http.HandlerFunc(groupHandler.UpdateMember)).Methods("PATCH")
	adminRouter.Handle("/groups/{id}/members/{userId}", http.HandlerFunc(groupHandler.RemoveMember)).Methods("DELETE")

	// Connections (Postgres, MySQL, MSSQL)
	connHandler := server.NewConnectionHandler(d.reg, d.cfg.EncryptionKey)
	adminRouter.Handle("/connections", http.HandlerFunc(connHandler.List)).Methods("GET")
	adminRouter.Handle("/connections", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.Create))).Methods("POST")
	adminRouter.Handle("/connections/test", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.TestRaw))).Methods("POST")
	adminRouter.Handle("/connections/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.Get))).Methods("GET")
	adminRouter.Handle("/connections/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.Update))).Methods("PUT")
	adminRouter.Handle("/connections/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.Delete))).Methods("DELETE")
	adminRouter.Handle("/connections/{id}/test", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.Test))).Methods("POST")
	adminRouter.Handle("/connections/{id}/introspect", http.HandlerFunc(connHandler.Introspect)).Methods("GET")
	adminRouter.Handle("/connections/{id}/generate-endpoints", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(connHandler.GenerateEndpoints))).Methods("POST")

	// Gateway Domains (Host-Based Routing)
	domainHandler := server.NewDomainHandler(d.reg)
	adminRouter.HandleFunc("/domains", domainHandler.List).Methods("GET")
	adminRouter.Handle("/domains", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(domainHandler.Create))).Methods("POST")
	adminRouter.Handle("/domains/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(domainHandler.Delete))).Methods("DELETE")

	// SQL Query Runner & Preview
	queryHandler := server.NewQueryHandler(d.reg, d.executorService)
	adminRouter.HandleFunc("/queries", queryHandler.List).Methods("GET")
	adminRouter.HandleFunc("/queries", queryHandler.Create).Methods("POST")
	adminRouter.HandleFunc("/queries/{id}", queryHandler.Get).Methods("GET")
	adminRouter.HandleFunc("/queries/{id}", queryHandler.Update).Methods("PUT")
	adminRouter.HandleFunc("/queries/{id}", queryHandler.Delete).Methods("DELETE")
	adminRouter.HandleFunc("/queries/preview", queryHandler.Preview).Methods("POST")

	// Services (Kong Logical Gateway Services)
	servicesHandler := server.NewServicesHandler(d.reg)
	adminRouter.HandleFunc("/services", servicesHandler.List).Methods("GET")
	adminRouter.Handle("/services", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(servicesHandler.Create))).Methods("POST")
	adminRouter.HandleFunc("/services/{id}", servicesHandler.Get).Methods("GET")
	adminRouter.Handle("/services/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(servicesHandler.Update))).Methods("PUT")
	adminRouter.Handle("/services/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(servicesHandler.Delete))).Methods("DELETE")
	adminRouter.HandleFunc("/services/{id}/routes", servicesHandler.Routes).Methods("GET")

	// Upstreams (Load-Balancing Pools & Health Checks)
	upstreamsHandler := server.NewUpstreamsHandler(d.reg)
	adminRouter.HandleFunc("/upstreams", upstreamsHandler.List).Methods("GET")
	adminRouter.Handle("/upstreams", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.Create))).Methods("POST")
	adminRouter.HandleFunc("/upstreams/{id}", upstreamsHandler.Get).Methods("GET")
	adminRouter.Handle("/upstreams/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.Update))).Methods("PUT")
	adminRouter.Handle("/upstreams/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.Delete))).Methods("DELETE")
	adminRouter.HandleFunc("/upstreams/{id}/health-check", upstreamsHandler.HealthCheckUpstream).Methods("POST")

	// Upstream Targets
	adminRouter.HandleFunc("/upstreams/{id}/targets", upstreamsHandler.ListTargets).Methods("GET")
	adminRouter.Handle("/upstreams/{id}/targets", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.CreateTarget))).Methods("POST")
	adminRouter.Handle("/targets/{targetId}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.UpdateTarget))).Methods("PUT")
	adminRouter.Handle("/targets/{targetId}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.DeleteTarget))).Methods("DELETE")
	adminRouter.Handle("/targets/{targetId}/toggle", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(upstreamsHandler.ToggleTarget))).Methods("POST")
	adminRouter.HandleFunc("/targets/{targetId}/health-check", upstreamsHandler.HealthCheckTarget).Methods("POST")

	// Routes Publishing (Canonical Kong name for Endpoints)
	endpointHandler := server.NewEndpointHandler(d.reg, d.executorService.GetEndpointCache())
	adminRouter.HandleFunc("/routes", endpointHandler.List).Methods("GET")
	adminRouter.HandleFunc("/routes", endpointHandler.Create).Methods("POST")
	adminRouter.HandleFunc("/routes/{id}", endpointHandler.Get).Methods("GET")
	adminRouter.HandleFunc("/routes/{id}", endpointHandler.Update).Methods("PUT")
	adminRouter.HandleFunc("/routes/{id}", endpointHandler.Delete).Methods("DELETE")

	// Legacy Endpoints Compatibility Aliases
	adminRouter.HandleFunc("/endpoints", endpointHandler.List).Methods("GET")
	adminRouter.HandleFunc("/endpoints", endpointHandler.Create).Methods("POST")
	adminRouter.HandleFunc("/endpoints/{id}", endpointHandler.Get).Methods("GET")
	adminRouter.HandleFunc("/endpoints/{id}", endpointHandler.Update).Methods("PUT")
	adminRouter.HandleFunc("/endpoints/{id}", endpointHandler.Delete).Methods("DELETE")

	// Consumers (Kong External Clients, Credentials, and Consumer Groups)
	consumersHandler := server.NewConsumersHandler(d.reg)
	adminRouter.HandleFunc("/consumers", consumersHandler.List).Methods("GET")
	adminRouter.Handle("/consumers", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.Create))).Methods("POST")
	adminRouter.HandleFunc("/consumers/{id}", consumersHandler.Get).Methods("GET")
	adminRouter.Handle("/consumers/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.Update))).Methods("PUT")
	adminRouter.Handle("/consumers/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.Delete))).Methods("DELETE")
	adminRouter.HandleFunc("/consumers/{id}/credentials", consumersHandler.ListCredentials).Methods("GET")
	adminRouter.Handle("/consumers/{id}/credentials", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.CreateCredential))).Methods("POST")
	adminRouter.Handle("/consumers/{id}/credentials/{credId}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.DeleteCredential))).Methods("DELETE")
	adminRouter.Handle("/consumers/{id}/credentials/{credId}/toggle", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.ToggleCredential))).Methods("POST")
	adminRouter.HandleFunc("/consumers/{id}/routes", consumersHandler.GetRoutes).Methods("GET")
	adminRouter.Handle("/consumers/{id}/routes", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.UpdateRoutes))).Methods("PUT")

	// Auth Plugins & Outbound Authentication (Inbound Plugins & Upstream Outbound Auth)
	authPluginsHandler := server.NewAuthPluginsHandler(d.reg, d.cfg.EncryptionKey)
	adminRouter.HandleFunc("/auth-plugins", authPluginsHandler.List).Methods("GET")
	adminRouter.Handle("/auth-plugins", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(authPluginsHandler.Create))).Methods("POST")
	adminRouter.HandleFunc("/auth-plugins/{id}", authPluginsHandler.Get).Methods("GET")
	adminRouter.Handle("/auth-plugins/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(authPluginsHandler.Update))).Methods("PUT")
	adminRouter.Handle("/auth-plugins/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(authPluginsHandler.Delete))).Methods("DELETE")

	// Upstream Outbound Auth
	adminRouter.HandleFunc("/services/{id}/outbound-auth", authPluginsHandler.GetOutboundAuth).Methods("GET")
	adminRouter.Handle("/services/{id}/outbound-auth", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(authPluginsHandler.SetOutboundAuth))).Methods("PUT", "POST")
	adminRouter.Handle("/services/{id}/outbound-auth", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(authPluginsHandler.DeleteOutboundAuth))).Methods("DELETE")

	// Consumer Groups
	adminRouter.HandleFunc("/consumer-groups", consumersHandler.ListGroups).Methods("GET")
	adminRouter.Handle("/consumer-groups", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.CreateGroup))).Methods("POST")
	adminRouter.HandleFunc("/consumer-groups/{id}", consumersHandler.GetGroup).Methods("GET")
	adminRouter.Handle("/consumer-groups/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.UpdateGroup))).Methods("PUT")
	adminRouter.Handle("/consumer-groups/{id}", d.authService.RequireRole("ADMIN", "SUPER_ADMIN")(http.HandlerFunc(consumersHandler.DeleteGroup))).Methods("DELETE")

	// Legacy External API Clients Compatibility Aliases
	apiClientHandler := server.NewAPIClientHandler(d.reg)
	adminRouter.Handle("/api-clients", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(apiClientHandler.List))).Methods("GET")
	adminRouter.Handle("/api-clients", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(apiClientHandler.Create))).Methods("POST")
	adminRouter.Handle("/api-clients/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(apiClientHandler.Get))).Methods("GET")
	adminRouter.Handle("/api-clients/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(apiClientHandler.Update))).Methods("PUT")
	adminRouter.Handle("/api-clients/{id}", d.authService.RequireRole("SUPER_ADMIN")(http.HandlerFunc(apiClientHandler.Delete))).Methods("DELETE")

	// Monitoring & Audit Logs
	dashboardHandler := monitoring.NewDashboardHandler(d.monitoringService)
	adminRouter.HandleFunc("/dashboard/stats", dashboardHandler.Stats).Methods("GET")
	adminRouter.HandleFunc("/monitoring/requests", dashboardHandler.Requests).Methods("GET")
	adminRouter.HandleFunc("/audit-logs", dashboardHandler.AuditLogs).Methods("GET")

	// GitOps Metadata Export/Import
	gitopsHandler := gitops.NewHandler(d.reg, d.publisher)
	adminRouter.HandleFunc("/metadata/export", gitopsHandler.Export).Methods("GET")
	adminRouter.HandleFunc("/metadata/import", gitopsHandler.Import).Methods("POST")

	// OpenAPI & Postman
	docsHandler := server.NewDocsHandler(d.openapiService, d.postmanService, d.reg)
	adminRouter.HandleFunc("/openapi.json", docsHandler.OpenAPI).Methods("GET")
	adminRouter.HandleFunc("/postman.json", docsHandler.Postman).Methods("GET")
	r.HandleFunc("/openapi.json", docsHandler.OpenAPI).Methods("GET")
	r.HandleFunc("/postman.json", docsHandler.Postman).Methods("GET")

	return r
}

// buildGUIRouter builds the dedicated Web Console GUI router (Port 8002)
func buildGUIRouter(d routerDeps, adminHandler http.Handler) *mux.Router {
	r := mux.NewRouter()

	// Optional IP allowlist middleware for GUI Console
	if d.cfg.AdminAllowedIPs != "" && d.cfg.AdminAllowedIPs != "*" {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				clientIP := iputil.ExtractClientIP(req)
				if !iputil.IsIPAllowed(clientIP, d.cfg.AdminAllowedIPs) {
					httputil.WriteJSONError(w, http.StatusForbidden, "IP not allowed to access Web Console UI")
					return
				}
				next.ServeHTTP(w, req)
			})
		})
	}

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"plane":   "control-plane",
			"service": "apirout-gui",
		})
	}).Methods("GET")

	// Transparently proxy /api/v1/* to admin handler for browser simplicity
	if adminHandler != nil {
		r.PathPrefix("/api/v1/").Handler(adminHandler)
	}

	// Serve React Vite Admin Console SPA
	spa := server.NewSPAHandler(d.cfg.StaticDir, "index.html")
	r.PathPrefix("/").Handler(spa)

	return r
}

// buildRouter builds combined router for legacy/standalone mode
func buildRouter(d routerDeps) *mux.Router {
	adminRouter := buildAdminAPIRouter(d)
	return buildGUIRouter(d, adminRouter)
}
