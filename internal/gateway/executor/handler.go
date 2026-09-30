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

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/iputil"
	"github.com/apirout/apirout/internal/core/models"
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
	proxyTransports  sync.Map // serviceID -> *http.Transport
}

// NewHandler creates a new dynamic API handler
func NewHandler(executor *Service, reader *registry.Reader, apiClientAuth *apiclientauth.Service, authEngine *auth.Engine, outboundInjector *auth.OutboundInjector) *Handler {
	return &Handler{
		executor:         executor,
		reader:           reader,
		apiClientAuth:    apiClientAuth,
		authEngine:       authEngine,
		outboundInjector: outboundInjector,
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

// Execute handles dynamic request matching and routing through Kong architectural flow:
// Request -> Route -> Consumer Auth -> Consumer Authz -> Service -> [Direct Host OR Upstream -> Target OR Database Query]
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path
	method := r.Method

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
		gwLatency := dur / 2
		upLatency := dur - gwLatency
		if dur < 2 {
			gwLatency = dur
			upLatency = 0
		}

		details, _ := json.Marshal(map[string]interface{}{
			"method":              method,
			"path":                path,
			"status":              statusCode,
			"status_code":         statusCode,
			"client_ip":           clientIP,
			"user_agent":          r.UserAgent(),
			"host":                r.Host,
			"size":                r.ContentLength,
			"route_name":          routeName,
			"service_name":        serviceName,
			"upstream_uri":        upstreamURI,
			"auth_type":           authType,
			"gateway_latency_ms":  gwLatency,
			"upstream_latency_ms": upLatency,
		})

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
	}()

	if !strings.HasPrefix(path, "/api/") {
		statusCode = http.StatusNotFound
		success = false
		httputil.WriteJSONError(w, http.StatusNotFound, "Not found")
		return
	}

	// 1. Route Matcher
	endpoint, err := h.reader.GetEndpointByPath(path)
	if err != nil {
		statusCode = http.StatusInternalServerError
		success = false
		errStr := "Database error"
		errorMsg = &errStr
		httputil.WriteJSONError(w, http.StatusInternalServerError, errStr)
		return
	}

	if endpoint == nil {
		statusCode = http.StatusNotFound
		success = false
		errStr := "Endpoint not found"
		errorMsg = &errStr
		httputil.WriteJSONError(w, http.StatusNotFound, errStr)
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
		httputil.WriteJSONError(w, http.StatusNotFound, errStr)
		return
	}

	// Match HTTP method
	if endpoint.Method != "*" && !strings.Contains(endpoint.Method, r.Method) {
		statusCode = http.StatusMethodNotAllowed
		success = false
		errStr := "Method not allowed"
		errorMsg = &errStr
		httputil.WriteJSONError(w, http.StatusMethodNotAllowed, errStr)
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
		httputil.WriteJSONError(w, http.StatusForbidden, errStr)
		return
	}

	// Client IP Whitelist Check (CIDR / direct IP / wildcard)
	if !iputil.IsIPAllowed(clientIP, endpoint.AllowedIPs) {
		statusCode = http.StatusForbidden
		success = false
		errStr := "Client IP address forbidden for this endpoint"
		errorMsg = &errStr
		httputil.WriteJSONError(w, http.StatusForbidden, errStr)
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
				httputil.WriteJSONError(w, http.StatusUnauthorized, errStr)
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
					httputil.WriteJSONError(w, http.StatusUnauthorized, errStr)
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
					httputil.WriteJSONError(w, http.StatusUnauthorized, errStr)
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
				httputil.WriteJSONError(w, http.StatusUnauthorized, errStr)
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
			httputil.WriteJSONError(w, http.StatusForbidden, errStr)
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
				if err != nil || len(targets) == 0 {
					statusCode = http.StatusServiceUnavailable
					success = false
					errStr := "All upstream targets are unavailable"
					errorMsg = &errStr
					httputil.WriteJSONError(w, http.StatusServiceUnavailable, errStr)
					return
				}

				algorithm := "round-robin"
				if upstream != nil && upstream.Algorithm != "" {
					algorithm = upstream.Algorithm
				}
				target := selectTarget(targets, algorithm, clientIP, r, *service.UpstreamID)
				if target != nil {
					upstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, target.Host, target.Port, service.Path)
					releaseConn := trackTargetConn(target.ID)
					defer releaseConn()
					h.reverseProxyRequest(w, r, endpoint, service, target.Host, target.Port, outboundAuth)
					return
				}
			}

			// Case B: Service points directly to a backend host/port
			if service.Host != "" && service.Protocol != "database" {
				upstreamURI = fmt.Sprintf("%s://%s:%d%s", service.Protocol, service.Host, service.Port, service.Path)
				h.reverseProxyRequest(w, r, endpoint, service, service.Host, service.Port, outboundAuth)
				return
			}
		}
	}

	// Case C: Database-backed SQL query execution (Preserves Instant API and SQL queries!)
	if serviceName == "" {
		serviceName = "Instant DB Query"
		upstreamURI = "internal://sql-engine"
	}

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

	result, err := h.executor.ExecuteEndpointByPath(path, params, session)
	if err != nil {
		statusCode = http.StatusInternalServerError
		success = false
		errStr := err.Error()
		errorMsg = &errStr
		httputil.WriteJSONError(w, http.StatusInternalServerError, errStr)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}

// getTransport retrieves or creates an HTTP transport with connection pooling for the backend service
func (h *Handler) getTransport(service *models.Service) *http.Transport {
	key := service.ID
	if key == "" {
		key = service.Name
	}
	if t, ok := h.proxyTransports.Load(key); ok {
		return t.(*http.Transport)
	}

	timeout := 60 * time.Second
	if service.ReadTimeout > 0 {
		timeout = time.Duration(service.ReadTimeout) * time.Millisecond
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: timeout,
	}

	actual, _ := h.proxyTransports.LoadOrStore(key, transport)
	return actual.(*http.Transport)
}

// reverseProxyRequest proxies an HTTP request to the resolved backend instance
func (h *Handler) reverseProxyRequest(w http.ResponseWriter, r *http.Request, endpoint *models.Endpoint, service *models.Service, backendHost string, backendPort int, outboundAuth *models.ServiceOutboundAuth) {
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
			if !strings.HasPrefix(subPath, "/") {
				subPath = "/" + subPath
			}
		}

		basePath := strings.TrimSuffix(service.Path, "/")
		req.URL.Path = basePath + subPath
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

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		httputil.WriteJSONError(rw, http.StatusBadGateway, fmt.Sprintf("Backend service %s unreachable: %v", service.Name, proxyErr))
	}

	proxy.ServeHTTP(w, r)
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
