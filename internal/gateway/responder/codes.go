package responder

// ErrorCode represents a canonical, machine-readable gateway error identifier.
type ErrorCode string

const (
	// Routing & Dispatch
	ErrCodeRouteNotFound   ErrorCode = "ROUTE_NOT_FOUND"
	ErrCodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"

	// Authentication & Authorization
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeInvalidKey   ErrorCode = "INVALID_API_KEY"
	ErrCodeExpiredToken ErrorCode = "EXPIRED_TOKEN"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"

	// Rate Limiting
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Upstream & Resilience
	ErrCodeUpstreamUnreachable ErrorCode = "UPSTREAM_UNREACHABLE"
	ErrCodeUpstreamTimeout     ErrorCode = "UPSTREAM_TIMEOUT"
	ErrCodeCircuitBreakerOpen  ErrorCode = "CIRCUIT_BREAKER_OPEN"
	ErrCodeNoHealthyTargets    ErrorCode = "NO_HEALTHY_TARGETS"
	ErrCodeBadGateway          ErrorCode = "BAD_GATEWAY"

	// Request Validation & Server
	ErrCodeBadRequest    ErrorCode = "BAD_REQUEST"
	ErrCodeNotFound      ErrorCode = "RECORD_NOT_FOUND"
	ErrCodeInternalError ErrorCode = "INTERNAL_ERROR"
)
