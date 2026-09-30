package registry

import (
	"encoding/json"
	"sort"
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string    `db:"id" json:"id"`
	EventType    string    `db:"event_type" json:"event_type"`
	UserID       *string   `db:"user_id" json:"user_id"`
	APIClientID  *string   `db:"api_client_id" json:"api_client_id"`
	EndpointID   *string   `db:"endpoint_id" json:"endpoint_id"`
	IPAddress    *string   `db:"ip_address" json:"ip_address"`
	Details      string    `db:"details" json:"details"`
	Success      bool      `db:"success" json:"success"`
	DurationMs   *int      `db:"duration_ms" json:"duration_ms"`
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
}

// CreateAuditLog creates a new audit log entry
func (r *Registry) CreateAuditLog(log *AuditLog) error {
	log.ID = generateID()
	log.Timestamp = time.Now()

	if log.Details == "" {
		log.Details = "{}"
	}

	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO audit_logs (id, event_type, user_id, api_client_id, endpoint_id, ip_address, details, success, duration_ms, error_message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		log.ID, log.EventType, log.UserID, log.APIClientID, log.EndpointID, log.IPAddress, log.Details,
		log.Success, log.DurationMs, log.ErrorMessage, log.Timestamp,
	)
	return err
}

// ListAuditLogs returns audit logs with optional filtering
func (r *Registry) ListAuditLogs(eventType string, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	var err error

	if limit <= 0 {
		limit = 100
	}

	// Explicitly select only mapped columns to avoid sqlx struct scan errors
	selectCols := "id, event_type, user_id, api_client_id, endpoint_id, ip_address, details, success, duration_ms, error_message, timestamp"

	if eventType != "" {
		err = r.db.Select(&logs,
			r.db.Rebind("SELECT "+selectCols+" FROM audit_logs WHERE event_type = ? ORDER BY timestamp DESC LIMIT ?"),
			eventType, limit,
		)
	} else {
		err = r.db.Select(&logs,
			r.db.Rebind("SELECT "+selectCols+" FROM audit_logs ORDER BY timestamp DESC LIMIT ?"),
			limit,
		)
	}
	return logs, err
}

// Metric represents a metric entry
type Metric struct {
	ID         string    `db:"id" json:"id"`
	EndpointID *string   `db:"endpoint_id" json:"endpoint_id"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
	DurationMs int       `db:"duration_ms" json:"duration_ms"`
	StatusCode int       `db:"status_code" json:"status_code"`
	Error      *string   `db:"error" json:"error"`
}

// CreateMetric creates a new metric entry
func (r *Registry) CreateMetric(metric *Metric) error {
	metric.ID = generateID()
	metric.Timestamp = time.Now()

	_, err := r.db.Exec(r.db.Rebind(`
		INSERT INTO metrics (id, endpoint_id, timestamp, duration_ms, status_code, error)
		VALUES (?, ?, ?, ?, ?, ?)`),
		metric.ID, metric.EndpointID, metric.Timestamp, metric.DurationMs, metric.StatusCode, metric.Error,
	)
	return err
}

// GetMetricsSummary returns aggregated metrics for dashboard
func (r *Registry) GetMetricsSummary(since time.Time) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	// Total requests
	var totalRequests int
	r.db.Get(&totalRequests, r.db.Rebind("SELECT COUNT(*) FROM metrics WHERE timestamp > ?"), since)
	summary["total_requests"] = totalRequests

	// Success/Error counts
	var successCount, errorCount int
	r.db.Get(&successCount, r.db.Rebind("SELECT COUNT(*) FROM metrics WHERE timestamp > ? AND status_code < 400"), since)
	r.db.Get(&errorCount, r.db.Rebind("SELECT COUNT(*) FROM metrics WHERE timestamp > ? AND status_code >= 400"), since)
	summary["success_count"] = successCount
	summary["error_count"] = errorCount

	// Error rate
	if totalRequests > 0 {
		summary["error_rate"] = float64(errorCount) / float64(totalRequests) * 100
	} else {
		summary["error_rate"] = 0.0
	}

	// Latency percentiles (approximate)
	var p50, p95 int
	r.db.Get(&p50, r.db.Rebind(`
		SELECT duration_ms FROM metrics WHERE timestamp > ?
		ORDER BY duration_ms LIMIT 1 OFFSET (SELECT COUNT(*)/2 FROM metrics WHERE timestamp > ?)`),
		since, since)
	r.db.Get(&p95, r.db.Rebind(`
		SELECT duration_ms FROM metrics WHERE timestamp > ?
		ORDER BY duration_ms LIMIT 1 OFFSET (SELECT COUNT(*)*95/100 FROM metrics WHERE timestamp > ?)`),
		since, since)
	summary["latency_p50_ms"] = p50
	summary["latency_p95_ms"] = p95

	return summary, nil
}

// TopEndpoint represents a top endpoint entry
type TopEndpoint struct {
	EndpointID    string  `db:"endpoint_id" json:"endpoint_id"`
	Method        *string `db:"method" json:"method"`
	Path          *string `db:"path" json:"path"`
	RequestCount  int     `db:"request_count" json:"request_count"`
	AvgDurationMs float64 `db:"avg_duration_ms" json:"avg_duration_ms"`
}

// GetTopEndpoints returns the most used endpoints with method and path
func (r *Registry) GetTopEndpoints(since time.Time, limit int) ([]TopEndpoint, error) {
	rows, err := r.db.Queryx(r.db.Rebind(`
		SELECT m.endpoint_id, e.method, e.path, COUNT(*) as request_count, AVG(m.duration_ms) as avg_duration_ms
		FROM metrics m
		LEFT JOIN endpoints e ON e.id = m.endpoint_id
		WHERE m.timestamp > ? AND m.endpoint_id IS NOT NULL
		GROUP BY m.endpoint_id, e.method, e.path
		ORDER BY request_count DESC
		LIMIT ?`), since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TopEndpoint
	for rows.Next() {
		var te TopEndpoint
		if err := rows.StructScan(&te); err != nil {
			return nil, err
		}
		results = append(results, te)
	}
	return results, nil
}

// TopUser represents a user's activity summary
type TopUser struct {
	UserID        *string   `db:"user_id" json:"user_id"`
	APIClientID   *string   `db:"api_client_id" json:"api_client_id"`
	Username      string    `json:"username"`
	TotalRequests int       `db:"total_requests" json:"total_requests"`
	SuccessCount  int       `db:"success_count" json:"success_count"`
	LastActive    time.Time `db:"last_active" json:"last_active"`
}

// GetTopUsers returns the most active users in the given period
func (r *Registry) GetTopUsers(since time.Time, limit int) ([]TopUser, error) {
	rows, err := r.db.Queryx(r.db.Rebind(`
		SELECT 
			user_id, 
			api_client_id, 
			COUNT(*) as total_requests, 
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
			MAX(timestamp) as last_active
		FROM audit_logs 
		WHERE timestamp > ? AND event_type = 'api_request'
		GROUP BY user_id, api_client_id
		ORDER BY total_requests DESC
		LIMIT ?`), since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TopUser
	for rows.Next() {
		var tu TopUser
		if err := rows.StructScan(&tu); err != nil {
			return nil, err
		}
		// Resolve username - check user_id first, then api_client_id
		if tu.UserID != nil && *tu.UserID != "" {
			if user, _ := r.GetUserByID(*tu.UserID); user != nil {
				tu.Username = user.Email
			} else {
				tu.Username = "Unknown User"
			}
		} else if tu.APIClientID != nil && *tu.APIClientID != "" {
			if client, _ := r.GetAPIClientByID(*tu.APIClientID); client != nil {
				tu.Username = "[API] " + client.Username
			} else {
				tu.Username = "Unknown API Client"
			}
		} else {
			tu.Username = "Anonymous"
		}
		results = append(results, tu)
	}
	return results, nil
}

// RecentRequest represents a single recent API request
type RecentRequest struct {
	ID           string    `db:"id" json:"id"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
	EventType    string    `db:"event_type" json:"event_type"`
	UserID       *string   `db:"user_id" json:"user_id"`
	APIClientID  *string   `db:"api_client_id" json:"api_client_id"`
	Username     string    `json:"username"`
	EndpointID   *string   `db:"endpoint_id" json:"endpoint_id"`
	IPAddress    *string   `db:"ip_address" json:"ip_address"`
	Details      string    `db:"details" json:"details"`
	Success      bool      `db:"success" json:"success"`
	DurationMs   *int      `db:"duration_ms" json:"duration_ms"`
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	// Parsed from details
	Method            string `json:"method"`
	Path              string `json:"path"`
	StatusCode        int    `json:"status_code"`
	ClientIP          string `json:"client_ip"`
	UserAgent         string `json:"user_agent"`
	Host              string `json:"host"`
	Size              int64  `json:"size"`
	RouteName         string `json:"route_name"`
	ServiceName       string `json:"service_name"`
	UpstreamURI       string `json:"upstream_uri"`
	AuthType          string `json:"auth_type"`
	GatewayLatencyMs  int    `json:"gateway_latency_ms"`
	UpstreamLatencyMs int    `json:"upstream_latency_ms"`
}

// GetRecentRequests returns recent API requests for dashboard
func (r *Registry) GetRecentRequests(limit int) ([]RecentRequest, error) {
	selectCols := "id, timestamp, event_type, user_id, api_client_id, endpoint_id, ip_address, details, success, duration_ms, error_message"
	rows, err := r.db.Queryx(r.db.Rebind(`
		SELECT `+selectCols+` FROM audit_logs 
		WHERE event_type = 'api_request'
		ORDER BY timestamp DESC
		LIMIT ?`), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RecentRequest
	for rows.Next() {
		var rr RecentRequest
		if err := rows.StructScan(&rr); err != nil {
			return nil, err
		}
		// Resolve username - check user_id first, then api_client_id
		if rr.UserID != nil && *rr.UserID != "" {
			if user, _ := r.GetUserByID(*rr.UserID); user != nil {
				rr.Username = user.Email
			} else {
				rr.Username = "Unknown User"
			}
		} else if rr.APIClientID != nil && *rr.APIClientID != "" {
			if client, _ := r.GetAPIClientByID(*rr.APIClientID); client != nil {
				rr.Username = "[API] " + client.Username
			} else {
				rr.Username = "Unknown API Client"
			}
		} else {
			rr.Username = "Anonymous"
		}
		// Parse details JSON for method, path, status, ip
		r.parseRequestDetails(&rr)
		// Map new IP column to ClientIP frontend prop 
		if rr.IPAddress != nil && *rr.IPAddress != "" {
			rr.ClientIP = *rr.IPAddress
		}
		results = append(results, rr)
	}
	return results, nil
}

// parseRequestDetails extracts method, path, status_code, client_ip from details JSON
func (r *Registry) parseRequestDetails(rr *RecentRequest) {
	// Details is JSON like: {"method":"GET","path":"/api/users","status_code":200,"client_ip":"127.0.0.1"}
	if rr.Details == "" || rr.Details == "{}" {
		return
	}

	type detailsJSON struct {
		Method            string `json:"method"`
		Path              string `json:"path"`
		StatusCode        int    `json:"status_code"`
		Status            int    `json:"status"`
		ClientIP          string `json:"client_ip"`
		UserAgent         string `json:"user_agent"`
		Host              string `json:"host"`
		Size              int64  `json:"size"`
		RouteName         string `json:"route_name"`
		ServiceName       string `json:"service_name"`
		UpstreamURI       string `json:"upstream_uri"`
		AuthType          string `json:"auth_type"`
		GatewayLatencyMs  int    `json:"gateway_latency_ms"`
		UpstreamLatencyMs int    `json:"upstream_latency_ms"`
	}

	var d detailsJSON
	if err := json.Unmarshal([]byte(rr.Details), &d); err == nil {
		rr.Method = d.Method
		rr.Path = d.Path
		if rr.StatusCode == 0 {
			if d.StatusCode != 0 {
				rr.StatusCode = d.StatusCode
			} else {
				rr.StatusCode = d.Status
			}
		}
		if rr.ClientIP == "" {
			rr.ClientIP = d.ClientIP
		}
		rr.UserAgent = d.UserAgent
		rr.Host = d.Host
		rr.Size = d.Size
		rr.RouteName = d.RouteName
		rr.ServiceName = d.ServiceName
		rr.UpstreamURI = d.UpstreamURI
		rr.AuthType = d.AuthType
		rr.GatewayLatencyMs = d.GatewayLatencyMs
		rr.UpstreamLatencyMs = d.UpstreamLatencyMs
	}
}

// TrafficTrend represents request count grouped by time
type TrafficTrend struct {
	Time         string `db:"time" json:"time"`
	RequestCount int    `db:"request_count" json:"request_count"`
}

// GetTrafficTrend returns the number of requests grouped by hour
func (r *Registry) GetTrafficTrend(since time.Time) ([]TrafficTrend, error) {
	// Initialize map with all 24 hours to ensure continuous chart
	counts := make(map[string]int)
	for i := 0; i <= 24; i++ {
		t := since.Add(time.Duration(i) * time.Hour)
		timeGroup := t.Format("2006-01-02T15:00:00Z")
		counts[timeGroup] = 0
	}

	// Fetch recent requests from table, avoiding SQLite string comparison timezone edge cases
	rows, err := r.db.Queryx(`SELECT timestamp FROM audit_logs WHERE event_type = ? ORDER BY timestamp DESC LIMIT 5000`, "api_request")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var raw interface{}
		if err := rows.Scan(&raw); err != nil {
			continue
		}

		var ts time.Time
		
		parseTimeStr := func(val string) time.Time {
			layouts := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02 15:04:05",
			}
			for _, layout := range layouts {
				if parsed, err := time.Parse(layout, val); err == nil {
					return parsed
				}
			}
			return time.Time{}
		}

		switch v := raw.(type) {
		case time.Time:
			ts = v
		case string:
			ts = parseTimeStr(v)
		case []byte:
			ts = parseTimeStr(string(v))
		}

		if ts.IsZero() {
			continue
		}

		// Only include events from our 24h window
		if ts.Before(since) {
			// Because we ordered by timestamp DESC, older ones will just be skipped here,
			// or we could theoretically break. Let's just continue to be safe if order anomalies exist.
			continue
		}

		// Group by hour
		timeGroup := ts.Format("2006-01-02T15:00:00Z")
		if _, exists := counts[timeGroup]; exists {
			counts[timeGroup]++
		}
	}

	var results []TrafficTrend
	for t, c := range counts {
		results = append(results, TrafficTrend{Time: t, RequestCount: c})
	}

	// Sort by time
	sort.Slice(results, func(i, j int) bool {
		return results[i].Time < results[j].Time
	})

	return results, nil
}
