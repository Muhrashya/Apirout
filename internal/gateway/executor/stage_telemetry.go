package executor

import (
	"encoding/json"
	"time"

	"github.com/apirout/apirout/internal/core/models"
	"github.com/apirout/apirout/internal/gateway/pipeline"
)

type telemetryStage struct {
	handler *Handler
}

func newTelemetryStage(h *Handler) *telemetryStage {
	return &telemetryStage{handler: h}
}

func (s *telemetryStage) Name() string { return "telemetry" }

func (s *telemetryStage) Execute(c *pipeline.Context) error {
	return nil
}

func (s *telemetryStage) Finish(c *pipeline.Context) {
	dur := int(time.Since(c.StartTime).Milliseconds())
	if c.Recorder.StatusCode != 0 {
		c.StatusCode = c.Recorder.StatusCode
	}
	if c.StatusCode >= 400 {
		c.Success = false
	}
	upLatency := c.UpstreamDurationMs
	gwLatency := dur - upLatency
	if gwLatency < 0 {
		gwLatency = 0
	}
	respSize := c.Request.ContentLength
	if c.Recorder.BytesWritten > 0 {
		respSize = c.Recorder.BytesWritten
	}

	endpointIDStr := ""
	if c.Endpoint != nil {
		endpointIDStr = c.Endpoint.ID
	}
	var epID *string
	if endpointIDStr != "" {
		epID = &endpointIDStr
	}

	details, _ := json.Marshal(map[string]interface{}{
		"method":              c.Method,
		"path":                c.Path,
		"status":              c.StatusCode,
		"status_code":         c.StatusCode,
		"client_ip":           c.ClientIP,
		"user_agent":          c.Request.UserAgent(),
		"host":                c.Request.Host,
		"size":                respSize,
		"route_name":          c.RouteName,
		"service_name":        c.ServiceName,
		"upstream_uri":        c.UpstreamURI,
		"auth_type":           c.AuthType,
		"gateway_latency_ms":  gwLatency,
		"upstream_latency_ms": upLatency,
	})

	if s.handler.reader != nil {
		go func() {
			_ = s.handler.reader.CreateAuditLog(&models.AuditLog{
				ID:           fmtID(),
				EventType:    "api_request",
				UserID:       c.UserID,
				APIClientID:  c.ConsumerID,
				EndpointID:   epID,
				IPAddress:    &c.ClientIP,
				Details:      string(details),
				DurationMs:   &dur,
				Success:      c.Success,
				ErrorMessage: c.ErrorMessage,
				Timestamp:    c.StartTime,
			})
		}()
	}
}
