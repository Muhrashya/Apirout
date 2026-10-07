package responder

// ErrorEnvelope defines the hybrid, backward-compatible error response shape emitted by the Gateway.
type ErrorEnvelope struct {
	Status    bool      `json:"status"` // Preserved for backward compatibility (always false on error)
	Message   string    `json:"message"`
	Error     string    `json:"error"`
	Code      ErrorCode `json:"code"`
	RequestID string    `json:"request_id"`
	Timestamp string    `json:"timestamp"`
	Details   any       `json:"details,omitempty"`
}
