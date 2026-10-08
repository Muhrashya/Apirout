package pipeline

import (
	"net/http"
	"time"

	"github.com/apirout/apirout/internal/core/httputil"
	"github.com/apirout/apirout/internal/core/models"
)

// ResponseRecorder wraps http.ResponseWriter to track status code and byte count
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode   int
	BytesWritten int64
}

// WriteHeader captures the HTTP status code
func (rec *ResponseRecorder) WriteHeader(code int) {
	rec.StatusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// Write captures written bytes
func (rec *ResponseRecorder) Write(b []byte) (int, error) {
	if rec.StatusCode == 0 {
		rec.StatusCode = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(b)
	rec.BytesWritten += int64(n)
	return n, err
}

// Flush supports http.Flusher if available
func (rec *ResponseRecorder) Flush() {
	if f, ok := rec.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Context carries state across the 4 stages of the execution pipeline
type Context struct {
	StartTime      time.Time
	Request        *http.Request
	Recorder       *ResponseRecorder
	ClientIP       string
	Path           string
	Method         string

	// Routing & Matching
	Endpoint       *models.Endpoint
	PathParams     map[string]string
	RouteName      string
	ServiceName    string
	UpstreamURI    string

	// Authentication & Identity
	AuthType       string
	ConsumerID     *string
	GroupID        string
	UserID         *string

	// Telemetry & Error state
	StatusCode         int
	Success            bool
	ErrorMessage       *string
	UpstreamDurationMs int
	GatewayDurationMs  int
	Handled            bool // Set to true when response writing is completely finished
}

// NewContext creates an initialized Context for an incoming HTTP request
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	rec := &ResponseRecorder{ResponseWriter: w, StatusCode: http.StatusOK}
	return &Context{
		StartTime:    time.Now(),
		Request:      r,
		Recorder:     rec,
		ClientIP:     httputil.GetClientIP(r),
		Path:         r.URL.Path,
		Method:       r.Method,
		StatusCode:   http.StatusOK,
		Success:      true,
		AuthType:     "none",
	}
}

// MarkHandled flags the request as finalized, halting downstream stages
func (c *Context) MarkHandled() {
	c.Handled = true
}

// SetError records an error state on the context
func (c *Context) SetError(status int, msg string) {
	c.StatusCode = status
	c.Success = false
	c.ErrorMessage = &msg
	c.Handled = true
}
