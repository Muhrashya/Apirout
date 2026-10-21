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
	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/core/router"
	"github.com/apirout/apirout/internal/core/upstream"
	"github.com/apirout/apirout/internal/gateway/apiclientauth"
	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/pipeline"
	"github.com/apirout/apirout/internal/gateway/registry"
	"github.com/apirout/apirout/internal/gateway/responder"
	"github.com/apirout/apirout/internal/gateway/targetpool"
	"github.com/google/uuid"
)

// Handler handles dynamic API requests for published routes and endpoints
type Handler struct {
	executor         *Service
	reader           *registry.Reader
	apiClientAuth    *apiclientauth.Service
	authEngine       *auth.Engine
	outboundInjector *auth.OutboundInjector
	instantHandler   *InstantCRUDHandler
	targetPool       targetpool.TargetPool
	rateLimiter      RateLimiter
	proxyTransports  sync.Map // serviceID -> *http.Transport
	router           atomic.Pointer[router.RadixRouter]
	circuitBreaker   *upstream.PassiveCircuitBreaker
	pipeline         *pipeline.Pipeline
	telemetry        *telemetryStage
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
		targetPool:       targetpool.New(),
		rateLimiter:      newTokenBucketLimiter(),
		circuitBreaker:   upstream.NewPassiveCircuitBreaker(3, 15*time.Second),
	}

	h.telemetry = newTelemetryStage(h)
	h.pipeline = pipeline.New(
		newIngressStage(h),
		newAuthStage(h),
		newUpstreamStage(h),
	)

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

// GetTargetPool returns the upstream target load-balancing pool
func (h *Handler) GetTargetPool() targetpool.TargetPool {
	return h.targetPool
}

// SetTargetPool configures the upstream target load-balancing pool
func (h *Handler) SetTargetPool(pool targetpool.TargetPool) {
	if pool != nil {
		h.targetPool = pool
	}
}

// GetRateLimiter returns the active rate limiter
func (h *Handler) GetRateLimiter() RateLimiter {
	return h.rateLimiter
}

// SetRateLimiter configures the active rate limiter
func (h *Handler) SetRateLimiter(rl RateLimiter) {
	if rl != nil {
		h.rateLimiter = rl
	}
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

// Execute handles dynamic request matching and routing through the Execution Pipeline
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	pCtx := pipeline.NewContext(w, r)
	defer h.telemetry.Finish(pCtx)

	_ = h.pipeline.Execute(pCtx)
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
		responder.Error(w, r, http.StatusBadGateway, responder.ErrCodeBadGateway, "Invalid backend URL")
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
		responder.HandleProxyError(rw, req, service.Name, proxyErr, targetID, h.circuitBreaker, upDuration, upStart)
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		return responder.ModifyProxyResponse(r, res, targetID, h.circuitBreaker, upDuration, upStart)
	}

	proxy.ServeHTTP(w, r)
	if upDuration != nil {
		*upDuration = int(time.Since(upStart).Milliseconds())
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

func isOriginAllowed(origin, allowedOrigins string) bool {
	allowedOrigins = strings.TrimSpace(allowedOrigins)
	if allowedOrigins == "" || allowedOrigins == "*" {
		return true
	}
	origins := strings.Split(allowedOrigins, ",")
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
		if strings.HasPrefix(o, "*.") {
			suffix := o[1:]
			if strings.HasSuffix(strings.ToLower(origin), strings.ToLower(suffix)) {
				return true
			}
		}
	}
	return false
}

func (h *Handler) isGatewayHostAllowed(host string) bool {
	if h.reader == nil {
		return true
	}
	// Always allow localhost, internal loopbacks, and direct IPs
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasPrefix(host, "127.") {
		return true
	}

	domains, err := h.reader.GetGatewayDomains()
	if err != nil || len(domains) == 0 {
		return true
	}

	hasCustomDomains := false
	for _, d := range domains {
		if !d.IsDefault && d.DomainName != "*" && d.DomainName != "localhost" {
			hasCustomDomains = true
			break
		}
	}
	if !hasCustomDomains {
		return true
	}

	for _, d := range domains {
		if isDomainAllowed(host, d.DomainName) {
			return true
		}
	}
	return false
}
