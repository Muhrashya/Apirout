package responder

// ErrorEnvelope defines the standardized, machine-readable gateway error response shape.
type ErrorEnvelope struct {
	Status    bool      `json:"status"` // Preserved for defensive client checks (always false on error)
	Message   string    `json:"message"`
	Code      ErrorCode `json:"code"`
	RequestID string    `json:"request_id"`
	Timestamp string    `json:"timestamp"`
	Details   any       `json:"details,omitempty"`
}
