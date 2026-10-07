package executor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	nethttputil "net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"errors"
	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/iputil"
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/router"
	"github.com/apirout/apirout/internal/core/upstream"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/registry"
	"github.com/google/uuid"
)

var (
	rrCounters  sync.Map // upstreamID -> *uint64
	activeConns sync.Map // targetID -> *int64
)

// Handler handles dynamic API requests for published routes and endpoints
type Handler struct {
	executor         *Service
	reader           *registry.Reader
	apiClientAuth    *apiclientauth.Service
	authEngine       *auth.Engine
	outboundInjector *auth.OutboundInjector
	instantHandler   *InstantCRUDHandler
	rateLimiter      *tokenBucketLimiter
	proxyTransports  sync.Map // serviceID -> *http.Transport
	router           atomic.Pointer[router.RadixRouter]
	circuitBreaker   *upstream.PassiveCircuitBreaker
}

// NewHandler creates a new dynamic API handler
func NewHandler(executor *Service, reader *registry.Reader, apiClientAuth *apiclientauth.Service, authEngine *auth.Engine, outboundInjector *auth.OutboundInjector, instantHandler *InstantCRUDHandler) *Handler {
	h := &Handler{
		executor:         executor,
		reader:           reader,
		apiClientAuth:    apiClientAuth,
		authEngine:       authEngine,
		outboundInjector: outboundInjector,
		instantHandler:   instantHandler,
		rateLimiter:      newTokenBucketLimiter(),
		circuitBreaker:   upstream.NewPassiveCircuitBreaker(3, 15*time.Second),
	}

	// Preload in-memory Radix router from active endpoints at boot
	r := router.NewRadixRouter()
	if reader != nil {
		if endpoints, err := reader.GetAllActiveEndpoints(); err == nil {
			r = router.BuildFromEndpoints(endpoints)
		}
	}
	h.router.Store(r)

	return h
}

// SetRouter atomically installs a new compiled RadixRouter
func (h *Handler) SetRouter(r *router.RadixRouter) {
	if r != nil {
		h.router.Store(r)
	}
}

// GetRouter returns the current active in-memory RadixRouter
func (h *Handler) GetRouter() *router.RadixRouter {
	return h.router.Load()
}

// ReloadRouterFromDB compiles and atomically swaps the router from registry database
func (h *Handler) ReloadRouterFromDB() error {
	if h.reader == nil {
		return errors.New("registry reader not initialized")
	}
	endpoints, err := h.reader.GetAllActiveEndpoints()
	if err != nil {
		return err
	}
	newRouter := router.BuildFromEndpoints(endpoints)
	h.SetRouter(newRouter)
	return nil
}

// GetCircuitBreaker returns the passive circuit breaker instance
func (h *Handler) GetCircuitBreaker() *upstream.PassiveCircuitBreaker {
	return h.circuitBreaker
}

// OnTargetHealthChanged updates the circuit breaker state based on Control Plane health events
func (h *Handler) OnTargetHealthChanged(targetID, action string) {
	if h.circuitBreaker == nil {
		return
	}
	if strings.EqualFold(action, "deleted") || strings.EqualFold(action, "unhealthy") {
		h.circuitBreaker.TripTarget(targetID)
	} else {
		h.circuitBreaker.ResetTarget(targetID)
	}
}

func extractTokenOrKey(r *http.Request) (token string, apiKey string) {
	// 1. Direct API Key header
	if key := r.Header.Get("X-API-Key"); key != "" {
		return "", key
	}

	// 2. Query param ?api_key=...
	if key := r.URL.Query().Get("api_key"); key != "" {
		return "", key
	}

	// 3. Authorization header: Bearer ak_... OR Bearer <jwt>
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			bearerVal := parts[1]
			if strings.HasPrefix(bearerVal, "ak_") {
				return "", bearerVal
			}
			return bearerVal, ""
		}
	}

	// 4. Legacy X-API-Token
	if apiTokenHeader := r.Header.Get("X-API-Token"); apiTokenHeader != "" {
		if strings.HasPrefix(apiTokenHeader, "ak_") {
			return "", apiTokenHeader
		}
		return apiTokenHeader, ""
	}

	return "", ""
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if rec.statusCode == 0 {
		rec.statusCode = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.bytesWritten += int64(n)
	return n, err
}

func (rec *statusRecorder) Flush() {
	if f, ok := rec.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rec *statusRecorder) CloseNotify() <-chan bool {
	if cn, ok := rec.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

type tokenBucketLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

func newTokenBucketLimiter() *tokenBucketLimiter {
	l := &tokenBucketLimiter{
		buckets: make(map[string]*tokenBucket),
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			l.cleanup()
		}
	}()
	return l
}

func (l *tokenBucketLimiter) Allow(key string, rpm int, burst int) bool {
	if rpm <= 0 {
		return true
	}
	if burst <= 0 {
		burst = rpm / 6
		if burst < 1 {
			burst = 1
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, exists := l.buckets[key]
	if !exists {
		l.buckets[key] = &tokenBucket{
			tokens:     float64(burst - 1),
			lastUpdate: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.lastUpdate = now
	ratePerSec := float64(rpm) / 60.0
	b.tokens += elapsed * ratePerSec
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

func (l *tokenBucketLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, b := range l.buckets {
		if b.lastUpdate.Before(cutoff) {
			delete(l.buckets, k)
		}
	}
}

func extractTableAndID(routePath, requestPath, servicePath string) (string, string) {
	var table, id string

	if servicePath != "" {
		trimmed := strings.Trim(servicePath, "/")
		if trimmed != "" {
			table = trimmed
		}
	}

	cleanRoute := strings.Trim(routePath, "/")
	cleanReq := strings.Trim(requestPath, "/")

	routeParts := strings.Split(cleanRoute, "/")
	reqParts := strings.Split(cleanReq, "/")

	if table == "" {
		for i := len(routeParts) - 1; i >= 0; i-- {
			part := routeParts[i]
			if !strings.HasPrefix(part, ":") && !strings.HasPrefix(part, "{") && part != "api" && part != "v1" && part != "v2" {
				table = part
				break
			}
		}
		if table == "" && len(routeParts) > 0 {
			table = routeParts[len(routeParts)-1]
		}
	}

	if len(reqParts) > len(routeParts) {
		id = reqParts[len(reqParts)-1]
	} else if len(routeParts) > 0 {
		lastRoutePart := routeParts[len(routeParts)-1]
		if strings.HasPrefix(lastRoutePart, ":") || strings.HasPrefix(lastRoutePart, "{") {
			if len(reqParts) == len(routeParts) {
				id = reqParts[len(reqParts)-1]
			}
		}
	}

	return table, id
}

func parseParams(r *http.Request) map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	if r.Body != nil && r.ContentLength > 0 && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var bodyMap map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&bodyMap); err == nil {
			for k, v := range bodyMap {
				params[k] = v
			}
		}
	}
	return params
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Execute(w, r)
}

// Execute handles dynamic request matching and routing through Kong architectural flow:
// Request -> Route -> Consumer Auth -> Consumer Authz -> Service -> [Direct Host OR Upstream -> Target OR Database Query/Instant CRUD]
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path
	method := r.Method

	rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	var upstreamDurationMs int = 0

	var endpointID *string
	var apiClientID *string
	var userID *string
	var statusCode = http.StatusOK
	var success = true
	var errorMsg *string
	var routeName = ""
	var serviceName = ""
	var upstreamURI = ""
	var authType = "none"
	clientIP := httputil.GetClientIP(r)

	defer func() {
		dur := int(time.Since(start).Milliseconds())
		if rec.statusCode != 0 {
			statusCode = rec.statusCode
		}
		if statusCode >= 400 {
			success = false
		}
		upLatency := upstreamDurationMs
		gwLatency := dur - upLatency
		if gwLatency < 0 {
			gwLatency = 0
		}
		respSize := r.ContentLength
		if rec.bytesWritten > 0 {
			respSize = rec.bytesWritten
		}

		details, _ := json.Marshal(map[string]interface{}{
			"method":              method,
			"path":                path,
			"status":              statusCode,
			"status_code":         statusCode,
			"client_ip":           clientIP,
			"user_agent":          r.UserAgent(),
			"host":                r.Host,
			"size":                respSize,
			"route_name":          routeName,
			"service_name":        serviceName,
			"upstream_uri":        upstreamURI,
			"auth_type":           authType,
			"gateway_latency_ms":  gwLatency,
			"upstream_latency_ms": upLatency,
		})

		if h.reader != nil {
			go func() {
				_ = h.reader.CreateAuditLog(&models.AuditLog{
					ID:           fmtID(),
					EventType:    "api_request",
					UserID:       userID,
					APIClientID:  apiClientID,
					EndpointID:   endpointID,
					IPAddress:    &clientIP,
					Details:      string(details),
					DurationMs:   &dur,
					Success:      success,
					ErrorMessage: errorMsg,
					Timestamp:    start,
				})
			}()
		}
	}()

	if !strings.HasPrefix(path, "/api/") {
		statusCode = http.StatusNotFound
		success = false
		httputil.WriteJSONError(rec, http.StatusNotFound, "Not found")
		return
	}

	// 1. In-Memory Radix Route Matcher (<1ms)
	var endpoint *models.Endpoint
	var pathParams map[string]string
	curRouter := h.router.Load()
	if curRouter != nil {
		res, matchErr := curRouter.Match(path, r.Method)
		if matchErr == nil && res != nil {
			endpoint = res.Endpoint
			pathParams = res.Params
		} else if errors.Is(matchErr, router.ErrMethodNotAllowed) {
			statusCode = http.StatusMethodNotAllowed
			success = false
			errStr := "Method not allowed"
			errorMsg = &errStr
			if len(res.AllowedMethods) > 0 {
				rec.Header().Set("Allow", strings.Join(res.AllowedMethods, ", "))
			}
			httputil.WriteJSONError(rec, http.StatusMethodNotAllowed, errStr)
			return
		}
	}

	// Fallback to registry reader if route not found in in-memory tree
	if endpoint == nil && h.reader != nil {
		ep, err := h.reader.GetEndpointByPath(path)
		if err != nil {
			statusCode = http.StatusInternalServerError
			success = false
			errStr := "Database error"
			errorMsg = &errStr
			httputil.WriteJSONError(rec, http.StatusInternalServerError, errStr)
			return
		}
		endpoint = ep
	}

	if endpoint == nil {
		statusCode = http.StatusNotFound
		success = false
		errStr := "Endpoint not found"
		errorMsg = &errStr
		httputil.WriteJSONError(rec, http.StatusNotFound, errStr)
		return
	}

	endpointID = &endpoint.ID
	routeName = endpoint.Name
	if endpoint.IsPublic {
		authType = "public"
	}

	if !endpoint.Enabled {
		statusCode = http.StatusNotFound
		success = false
		errStr := "Endpoint disabled"
		errorMsg = &errStr
		httputil.WriteJSONError(rec, http.StatusNotFound, errStr)
		return
	}

	// Match HTTP method (for fallback paths)
	if endpoint.Method != "*" && endpoint.Method != "ALL" && endpoint.Method != "ANY" && endpoint.Method != "" && !strings.Contains(endpoint.Method, r.Method) {
		statusCode = http.StatusMethodNotAllowed
		success = false
		errStr := "Method not allowed"
		errorMsg = &errStr
		rec.Header().Set("Allow", endpoint.Method)
		httputil.WriteJSONError(rec, http.StatusMethodNotAllowed, errStr)
		return
	}

	// Host-based / Domain Routing check
	host := r.Host
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		host = xfh
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	if !isDomainAllowed(host, endpoint.AllowedDomains) {
		statusCode = http.StatusForbidden
		success = false
		errStr := "Domain not authorized to access this endpoint"
		errorMsg = &errStr
		httputil.WriteJSONError(rec, http.StatusForbidden, errStr)
		return
	}

	// Client IP Whitelist Check (CIDR / direct IP / wildcard)
	if !iputil.IsIPAllowed(clientIP, endpoint.AllowedIPs) {
		statusCode = http.StatusForbidden
		success = false
		errStr := "Client IP address forbidden for this endpoint"
		errorMsg = &errStr
		httputil.WriteJSONError(rec, http.StatusForbidden, errStr)
		return
	}

	// 2. Consumer Authentication & Authorization Check
	var session *SessionContext

	if !endpoint.IsPublic {
		var consumerID string
		var groupID string
		var authResolved bool

		// 2a. Try new auth plugin engine first (if plugins are configured)
		if h.authEngine != nil && h.authEngine.HasPlugins(endpoint.ID, endpoint.ServiceID) {
			authResult, err := h.authEngine.Authenticate(r, endpoint, endpoint.ServiceID)
			if err != nil {
				statusCode = http.StatusUnauthorized
				success = false
				errStr := err.Error()
				errorMsg = &errStr
				httputil.WriteJSONError(rec, http.StatusUnauthorized, errStr)
				return
			}
			if authResult != nil {
				consumerID = authResult.ConsumerID
				groupID = authResult.GroupID
				authType = authResult.AuthType
				authResolved = true
			}
		}

		// 2b. Fallback to legacy auth if no plugin resolved the identity
		if !authResolved {
			jwtToken, rawAPIKey := extractTokenOrKey(r)

			if rawAPIKey != "" {
				// Authenticate using API Key credential
				consumer, _, err := h.reader.ValidateAPIKey(rawAPIKey)
				if err != nil {
					statusCode = http.StatusUnauthorized
					success = false
					errStr := "Invalid API key: " + err.Error()
					errorMsg = &errStr
					httputil.WriteJSONError(rec, http.StatusUnauthorized, errStr)
					return
				}
				consumerID = consumer.ID
				authType = "key-auth"
				if consumer.GroupID != nil {
					groupID = *consumer.GroupID
				}
			} else if jwtToken != "" {
				// Authenticate using JWT Bearer token
				claims, err := h.apiClientAuth.ValidateToken(jwtToken)
				if err != nil {
					statusCode = http.StatusUnauthorized
					success = false
					errStr := "Invalid or expired token"
					errorMsg = &errStr
					httputil.WriteJSONError(rec, http.StatusUnauthorized, errStr)
					return
				}
				consumerID = claims.APIClientID
				groupID = claims.GroupID
				authType = "jwt"
			} else {
				statusCode = http.StatusUnauthorized
				success = false
				errStr := "Authentication required (provide X-API-Key or Authorization Bearer token)"
				errorMsg = &errStr
				httputil.WriteJSONError(rec, http.StatusUnauthorized, errStr)
				return
			}
		}

		// Authorization Check: Verify Consumer has access to this Route
		allowed, err := h.reader.IsAPIClientAllowedForEndpoint(consumerID, endpoint.ID)
		if err != nil || !allowed {
			statusCode = http.StatusForbidden
			success = false
			errStr := "Access to this endpoint is forbidden for consumer"
			errorMsg = &errStr
			httputil.WriteJSONError(rec, http.StatusForbidden, errStr)
			return
		}

		apiClientID = &consumerID
		session = &SessionContext{
			SubjectID:   consumerID,
			SubjectType: "api_client",
			APIClientID: consumerID,
			GroupID:     groupID,
		}
	}

	// Rate Limiting Check
	if endpoint.RateLimitEnabled && endpoint.RateLimitRPM > 0 {
		rateKey := fmt.Sprintf("route:%s:ip:%s", endpoint.ID, clientIP)
		if apiClientID != nil {
			rateKey = fmt.Sprintf("route:%s:client:%s", endpoint.ID, *apiClientID)
		}
		if !h.rateLimiter.Allow(rateKey, endpoint.RateLimitRPM, endpoint.RateLimitBurst) {
			statusCode = http.StatusTooManyRequests
			success = false
			errStr := "Route rate limit exceeded"
			errorMsg = &errStr
			rec.Header().Set("Retry-After", "60")
			httputil.WriteJSONError(rec, http.StatusTooManyRequests, errStr)
			return
		}
	}

	// Consumer Group Rate Limiting Check
	if session != nil && session.GroupID != "" {
		group, err := h.reader.GetConsumerGroupByID(session.GroupID)
		if err == nil && group != nil && group.RateLimitRPM > 0 {
			groupKey := fmt.Sprintf("group:%s:client:%s", session.GroupID, session.APIClientID)
			if !h.rateLimiter.Allow(groupKey, group.RateLimitRPM, group.RateLimitBurst) {
				statusCode = http.StatusTooManyRequests
				success = false
				errStr := "Consumer group rate limit exceeded"
				errorMsg = &errStr
				rec.Header().Set("Retry-After", "60")
				httputil.WriteJSONError(rec, http.StatusTooManyRequests, errStr)
				return
			}
		}
	}

	// 3. Service Resolution & Gateway Execution
	var outboundAuth *models.ServiceOutboundAuth
	if endpoint.ServiceID != nil && *endpoint.ServiceID != "" {
		service, err := h.reader.GetServiceByID(*endpoint.ServiceID)
		if err == nil && service != nil && service.Enabled {
			serviceName = service.Name

			// Load outbound auth config for this service
			outboundAuth, _ = h.reader.GetOutboundAuth(service.ID)

			// Case A: Service routes to an Upstream pool (Load-balanced Targets)
			if service.UpstreamID != nil && *service.UpstreamID != "" {
				upstream, _ := h.reader.GetUpstreamByID(*service.UpstreamID)
				targets, err := h.reader.GetHealthyTargetsByUpstreamID(*service.UpstreamID)
				availableTargets := targets
				if h.circuitBreaker != nil {
					avail, _ := h.circuitBreaker.FilterAvailable(targets)
					availableTargets = avail
				}
				if err != nil || len(availableTargets) == 0 {
					statusCode = http.StatusServiceUnavailable
					success = false
					errStr := "no healthy upstream targets available"
					errorMsg = &errStr
					httputil.WriteJSONError(rec, http.StatusServiceUnavailable, errStr)
					return
				}

				algorithm := "round-robin"
				if upstream != nil && upstream.Algorithm != "" {
					algorithm = upstream.Algorithm
				}
				target := selectTarget(availableTargets, algorithm, clientIP, r, *service.UpstreamID)
				if target != nil {
					upstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, target.Host, target.Port, service.Path)
					releaseConn := trackTargetConn(target.ID)
					defer releaseConn()
					h.reverseProxyRequest(rec, r, endpoint, service, target.Host, target.Port, target.ID, outboundAuth, &upstreamDurationMs)
					return
				}
			}

			// Case B: Service points directly to a backend host/port
			if service.Host != "" && service.Protocol != "database" {
				upstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, service.Host, service.Port, service.Path)
				h.reverseProxyRequest(rec, r, endpoint, service, service.Host, service.Port, "", outboundAuth, &upstreamDurationMs)
				return
			}

			// Case C: Service routes to a Database Connection (Instant REST CRUD or SQL API)
			if (service.ConnectionID != nil && *service.ConnectionID != "") || service.Protocol == "database" {
				connID := ""
				if service.ConnectionID != nil {
					connID = *service.ConnectionID
				}
				if connID == "" {
					connID = service.ID
				}
				upstreamURI = fmt.Sprintf("database://%s/%s", connID, service.Path)

				// C1: If route has an explicit SQL query attached, execute that query
				if endpoint.QueryID != nil && *endpoint.QueryID != "" {
					params := parseParams(r)
					for k, v := range pathParams {
						params[k] = v
					}
					upStart := time.Now()
					result, err := h.executor.ExecuteEndpointByPath(path, params, session)
					upstreamDurationMs = int(time.Since(upStart).Milliseconds())
					if err != nil {
						statusCode = http.StatusInternalServerError
						success = false
						errStr := err.Error()
						errorMsg = &errStr
						httputil.WriteJSONError(rec, http.StatusInternalServerError, errStr)
						return
					}
					httputil.WriteJSON(rec, http.StatusOK, result)
					return
				}

				// C2: Route performs Instant REST table CRUD on the connected database
				if h.instantHandler != nil {
					table, id := extractTableAndID(endpoint.Path, r.URL.Path, service.Path)
					upStart := time.Now()
					h.instantHandler.ExecuteTableCRUD(rec, r, connID, table, id)
					upstreamDurationMs = int(time.Since(upStart).Milliseconds())
					statusCode = rec.statusCode
					if statusCode >= 400 {
						success = false
					}
					return
				}
			}
		}
	}

	// Case D: Endpoint with QueryID but no Service (Standalone SQL Query)
	if endpoint.QueryID != nil && *endpoint.QueryID != "" {
		if serviceName == "" {
			serviceName = "Standalone SQL Query"
			upstreamURI = "internal://sql-engine"
		}
		params := parseParams(r)
		for k, v := range pathParams {
			params[k] = v
		}
		upStart := time.Now()
		result, err := h.executor.ExecuteEndpointByPath(path, params, session)
		upstreamDurationMs = int(time.Since(upStart).Milliseconds())
		if err != nil {
			statusCode = http.StatusInternalServerError
			success = false
			errStr := err.Error()
			errorMsg = &errStr
			httputil.WriteJSONError(rec, http.StatusInternalServerError, errStr)
			return
		}
		httputil.WriteJSON(rec, http.StatusOK, result)
		return
	}

	// Case E: Route has neither a valid Service nor a Query
	statusCode = http.StatusBadRequest
	success = false
	errStr := "Route is not configured with an active Service backend, Database connection, or SQL query"
	errorMsg = &errStr
	httputil.WriteJSONError(rec, http.StatusBadRequest, errStr)
}

// getTransport retrieves or creates an HTTP transport with connection pooling and configurable timeouts for the backend service
func (h *Handler) getTransport(service *models.Service) *http.Transport {
	key := service.ID
	if key == "" {
		key = service.Name
	}
	if t, ok := h.proxyTransports.Load(key); ok {
		return t.(*http.Transport)
	}

	readTimeout := 60 * time.Second
	if service.ReadTimeout > 0 {
		readTimeout = time.Duration(service.ReadTimeout) * time.Millisecond
	}

	connectTimeout := 10 * time.Second
	if service.ConnectTimeout > 0 {
		connectTimeout = time.Duration(service.ConnectTimeout) * time.Millisecond
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   connectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: readTimeout,
	}

	actual, _ := h.proxyTransports.LoadOrStore(key, transport)
	return actual.(*http.Transport)
}

// reverseProxyRequest proxies an HTTP request to the resolved backend instance with accurate latency measurement
func (h *Handler) reverseProxyRequest(w http.ResponseWriter, r *http.Request, endpoint *models.Endpoint, service *models.Service, backendHost string, backendPort int, targetID string, outboundAuth *models.ServiceOutboundAuth, upDuration *int) {
	protocol := service.Protocol
	if protocol == "" || protocol == "database" {
		protocol = "http"
	}

	targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", protocol, backendHost, backendPort))
	if err != nil {
		httputil.WriteJSONError(w, http.StatusBadGateway, "Invalid backend URL")
		return
	}

	proxy := nethttputil.NewSingleHostReverseProxy(targetURL)

	// Capture outbound injector reference for closure
	outboundInjector := h.outboundInjector
	outAuth := outboundAuth

	// Custom Director to handle strip_path, preserve_host, headers, and outbound auth
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Path resolution
		subPath := r.URL.Path
		if endpoint.StripPath {
			subPath = strings.TrimPrefix(subPath, endpoint.Path)
			if subPath != "" && !strings.HasPrefix(subPath, "/") {
				subPath = "/" + subPath
			}
		}

		basePath := strings.TrimSuffix(service.Path, "/")
		if basePath != "" && (subPath == "" || subPath == "/") {
			req.URL.Path = basePath
		} else if basePath != "" {
			req.URL.Path = basePath + subPath
		} else {
			if subPath == "" {
				subPath = "/"
			}
			req.URL.Path = subPath
		}
		req.URL.RawQuery = r.URL.RawQuery

		// Host header
		if endpoint.PreserveHost {
			req.Host = r.Host
		} else {
			req.Host = fmt.Sprintf("%s:%d", backendHost, backendPort)
		}

		// Gateway identification headers
		clientIP := httputil.GetClientIP(r)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Forwarded-Proto", protocol)
		req.Header.Set("X-Gateway-Service", service.Name)
		req.Header.Set("X-Gateway-Route", endpoint.Name)

		// Outbound Authentication Injection
		if outboundInjector != nil && outAuth != nil {
			_ = outboundInjector.InjectAuth(req, outAuth)
		}
	}

	// Use pooled transport with keep-alive
	proxy.Transport = h.getTransport(service)

	upStart := time.Now()
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		if upDuration != nil {
			*upDuration = int(time.Since(upStart).Milliseconds())
		}
		if h.circuitBreaker != nil && targetID != "" {
			h.circuitBreaker.RecordFailure(targetID)
		}
		httputil.WriteJSONError(rw, http.StatusBadGateway, fmt.Sprintf("Backend service %s unreachable: %v", service.Name, proxyErr))
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		if h.circuitBreaker != nil && targetID != "" {
			if res.StatusCode >= 500 {
				h.circuitBreaker.RecordFailure(targetID)
			} else {
				h.circuitBreaker.RecordSuccess(targetID)
			}
		}
		return nil
	}

	proxy.ServeHTTP(w, r)
	if upDuration != nil {
		*upDuration = int(time.Since(upStart).Milliseconds())
	}
}

// selectTarget chooses a target instance using the specified Upstream algorithm
func selectTarget(targets []models.Target, algorithm string, clientIP string, r *http.Request, upstreamID string) *models.Target {
	if len(targets) == 0 {
		return nil
	}
	if len(targets) == 1 {
		return &targets[0]
	}

	switch strings.ToLower(algorithm) {
	case "round-robin":
		val, _ := rrCounters.LoadOrStore(upstreamID, new(uint64))
		counter := val.(*uint64)
		idx := atomic.AddUint64(counter, 1) % uint64(len(targets))
		return &targets[idx]

	case "consistent-hashing":
		hashKey := clientIP
		if hVal := r.Header.Get("X-User-ID"); hVal != "" {
			hashKey = hVal
		} else if hVal := r.Header.Get("X-Consumer-ID"); hVal != "" {
			hashKey = hVal
		}
		var h uint32
		for i := 0; i < len(hashKey); i++ {
			h = 31*h + uint32(hashKey[i])
		}
		idx := int(h) % len(targets)
		if idx < 0 {
			idx = -idx
		}
		return &targets[idx]

	case "least-connections":
		var minConn int64 = 1<<62 - 1
		var selected *models.Target = &targets[0]
		for i := range targets {
			c, _ := activeConns.LoadOrStore(targets[i].ID, new(int64))
			count := atomic.LoadInt64(c.(*int64))
			if count < minConn {
				minConn = count
				selected = &targets[i]
			}
		}
		return selected

	default: // round-robin fallback
		val, _ := rrCounters.LoadOrStore(upstreamID, new(uint64))
		counter := val.(*uint64)
		idx := atomic.AddUint64(counter, 1) % uint64(len(targets))
		return &targets[idx]
	}
}

func trackTargetConn(targetID string) func() {
	val, _ := activeConns.LoadOrStore(targetID, new(int64))
	counter := val.(*int64)
	atomic.AddInt64(counter, 1)
	return func() {
		atomic.AddInt64(counter, -1)
	}
}

func fmtID() string {
	return uuid.New().String()
}

func isDomainAllowed(host, allowedDomains string) bool {
	allowedDomains = strings.TrimSpace(allowedDomains)
	if allowedDomains == "" || allowedDomains == "*" {
		return true
	}
	domains := strings.Split(allowedDomains, ",")
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if d == "*" || strings.EqualFold(d, host) {
			return true
		}
		if strings.HasPrefix(d, "*.") {
			suffix := d[1:]
			if strings.HasSuffix(strings.ToLower(host), strings.ToLower(suffix)) {
				return true
			}
		}
	}
	return false
}
