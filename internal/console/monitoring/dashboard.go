package monitoring

import (
	"time"

	"github.com/apirout/apirout/internal/console/registry"
)

// Service handles monitoring and dashboard metrics
type Service struct {
	registry  *registry.Registry
	startTime time.Time
}

// NewService creates a new monitoring service
func NewService(reg *registry.Registry) *Service {
	return &Service{
		registry:  reg,
		startTime: time.Now(),
	}
}

// GetDashboardStats returns dashboard statistics
func (s *Service) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Uptime
	stats["uptime_seconds"] = int(time.Since(s.startTime).Seconds())
	stats["started_at"] = s.startTime.Format(time.RFC3339)

	// Get metrics from last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	metricsSummary, err := s.registry.GetMetricsSummary(since)
	if err != nil {
		return nil, err
	}

	for k, v := range metricsSummary {
		stats[k] = v
	}

	// Top endpoints
	topEndpoints, err := s.registry.GetTopEndpoints(since, 10)
	if err != nil {
		return nil, err
	}
	stats["top_endpoints"] = topEndpoints

	// Top users (24h)
	topUsers, err := s.registry.GetTopUsers(since, 6)
	if err != nil {
		// Non-fatal, just log and continue
		topUsers = []registry.TopUser{}
	}
	stats["top_users"] = topUsers

	// Recent requests
	recentRequests, err := s.registry.GetRecentRequests(10)
	if err != nil {
		// Non-fatal, just log and continue
		recentRequests = []registry.RecentRequest{}
	}
	stats["recent_requests"] = recentRequests

	// Traffic trend (24h)
	trafficTrend, err := s.registry.GetTrafficTrend(since)
	if err != nil {
		trafficTrend = []registry.TrafficTrend{}
	}
	stats["traffic_trend"] = trafficTrend

	// Connection health summary
	connections, err := s.registry.ListConnectionsWithHealth()
	if err != nil {
		return nil, err
	}

	healthyCount := 0
	unhealthyCount := 0
	for _, conn := range connections {
		// Handle both bool (fixed manually) and int64 (raw from SQLite) just in case
		if success, ok := conn["last_success"].(bool); ok && success {
			healthyCount++
		} else if success, ok := conn["last_success"].(int64); ok && success == 1 {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	stats["connections_healthy"] = healthyCount
	stats["connections_unhealthy"] = unhealthyCount
	stats["connections_total"] = len(connections)

	return stats, nil
}

// GetAuditLogs returns audit logs
func (s *Service) GetAuditLogs(eventType string, limit int) ([]registry.AuditLog, error) {
	return s.registry.ListAuditLogs(eventType, limit)
}

// GetAuditLogsPaginated returns filtered and paginated administrative audit logs
func (s *Service) GetAuditLogsPaginated(eventType, search string, page, limit int) ([]registry.AuditLog, int, error) {
	return s.registry.ListAuditLogsPaginated(eventType, search, page, limit)
}

// GetRequests returns parsed API request logs for Kong-style explorer
func (s *Service) GetRequests(limit int) ([]registry.RecentRequest, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.registry.GetRecentRequests(limit)
}

// GetRequestsPaginated returns filtered and paginated API request logs
func (s *Service) GetRequestsPaginated(filter registry.RequestFilter, page, limit int) ([]registry.RecentRequest, int, error) {
	return s.registry.GetRecentRequestsPaginated(filter, page, limit)
}

// StartRetentionPruner starts a background ticker that periodically prunes stale audit logs
func (s *Service) StartRetentionPruner(retentionDays int, interval time.Duration) func() {
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	// Run initial prune asynchronously
	go func() {
		if retentionDays > 0 {
			_, _ = s.registry.PruneAuditLogs(retentionDays)
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				if retentionDays > 0 {
					_, _ = s.registry.PruneAuditLogs(retentionDays)
				}
			}
		}
	}()

	return func() {
		close(done)
	}
}

