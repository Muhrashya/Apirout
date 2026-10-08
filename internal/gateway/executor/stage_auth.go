package executor

import (
	"fmt"
	"net/http"

	"github.com/apirout/apirout/internal/gateway/auth"
	"github.com/apirout/apirout/internal/gateway/pipeline"
	"github.com/apirout/apirout/internal/gateway/responder"
)

type authStage struct {
	handler *Handler
}

func newAuthStage(h *Handler) *authStage {
	return &authStage{handler: h}
}

func (s *authStage) Name() string { return "auth" }

func (s *authStage) Execute(c *pipeline.Context) error {
	endpoint := c.Endpoint
	if endpoint == nil {
		return nil
	}

	r := c.Request
	rec := c.Recorder

	if endpoint.IsPublic {
		c.AuthType = "public"
	} else {
		var consumerID string
		var groupID string

		// Authenticate inbound request via deep auth.Engine (plugins OR resolution with fallback)
		if s.handler.authEngine != nil {
			authResult, err := s.handler.authEngine.Authenticate(r, endpoint, endpoint.ServiceID)
			if err != nil {
				errStr := err.Error()
				c.SetError(http.StatusUnauthorized, errStr)
				if authErr, ok := err.(*auth.Error); ok {
					c.StatusCode = authErr.StatusCode
					responder.Error(rec, r, authErr.StatusCode, authErr.Code, authErr.Message)
				} else {
					responder.Error(rec, r, http.StatusUnauthorized, responder.ErrCodeUnauthorized, errStr)
				}
				return nil
			}
			if authResult != nil {
				consumerID = authResult.ConsumerID
				groupID = authResult.GroupID
				c.AuthType = authResult.AuthType
			}
		}

		// Authorization Check: Verify Consumer has access to this Route
		if s.handler.reader != nil {
			allowed, err := s.handler.reader.IsAPIClientAllowedForEndpoint(consumerID, endpoint.ID)
			if err != nil || !allowed {
				c.SetError(http.StatusForbidden, "Access to this endpoint is forbidden for consumer")
				responder.Error(rec, r, http.StatusForbidden, responder.ErrCodeForbidden, "Access to this endpoint is forbidden for consumer")
				return nil
			}
		}

		c.ConsumerID = &consumerID
		c.GroupID = groupID
	}

	// Rate Limiting Check
	if endpoint.RateLimitEnabled && endpoint.RateLimitRPM > 0 {
		rateKey := fmt.Sprintf("route:%s:ip:%s", endpoint.ID, c.ClientIP)
		if c.ConsumerID != nil {
			rateKey = fmt.Sprintf("route:%s:client:%s", endpoint.ID, *c.ConsumerID)
		}
		if !s.handler.rateLimiter.Allow(rateKey, endpoint.RateLimitRPM, endpoint.RateLimitBurst) {
			c.SetError(http.StatusTooManyRequests, "Route rate limit exceeded")
			responder.Error(rec, r, http.StatusTooManyRequests, responder.ErrCodeRateLimitExceeded, "Route rate limit exceeded", map[string]string{"Retry-After": "60"})
			return nil
		}
	}

	// Consumer Group Rate Limiting Check
	if c.GroupID != "" && s.handler.reader != nil {
		group, err := s.handler.reader.GetConsumerGroupByID(c.GroupID)
		if err == nil && group != nil && group.RateLimitRPM > 0 && c.ConsumerID != nil {
			groupKey := fmt.Sprintf("group:%s:client:%s", c.GroupID, *c.ConsumerID)
			if !s.handler.rateLimiter.Allow(groupKey, group.RateLimitRPM, group.RateLimitBurst) {
				c.SetError(http.StatusTooManyRequests, "Consumer group rate limit exceeded")
				responder.Error(rec, r, http.StatusTooManyRequests, responder.ErrCodeRateLimitExceeded, "Consumer group rate limit exceeded", map[string]string{"Retry-After": "60"})
				return nil
			}
		}
	}

	return nil
}
