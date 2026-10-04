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

// GetMetricsSummary returns aggregated metrics for dashboard from authoritative audit_logs
func (r *Registry) GetMetricsSummary(since time.Time) (map[string]interface{}, error) {
	summary := make(map[string]interface{})

	type requestRow struct {
		DurationMs *int   `db:"duration_ms"`
		Success    bool   `db:"success"`
		Details    string `db:"details"`
	}

	query := r.db.Rebind(`
		SELECT duration_ms, success, details 
		FROM audit_logs 
		WHERE event_type = 'api_request' AND timestamp > ?`)

	rows, err := r.db.Queryx(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalRequests := 0
	successCount := 0
	errorCount := 0
	status2xx := 0
	status3xx := 0
	status4xx := 0
	status5xx := 0
	var durations []int
	var totalDuration int64 = 0

	for rows.Next() {
		var row requestRow
		if err := rows.StructScan(&row); err != nil {
			continue
		}
		totalRequests++

		if row.Success {
			successCount++
		} else {
			errorCount++
		}

		if row.DurationMs != nil && *row.DurationMs >= 0 {
			dur := *row.DurationMs
			durations = append(durations, dur)
			totalDuration += int64(dur)
		}

		// Extract status code from details
		statusCode := 200
		if row.Details != "" && row.Details != "{}" {
			var d struct {
				StatusCode int `json:"status_code"`
				Status     int `json:"status"`
			}
			if err := json.Unmarshal([]byte(row.Details), &d); err == nil {
				if d.StatusCode != 0 {
					statusCode = d.StatusCode
				} else if d.Status != 0 {
					statusCode = d.Status
				}
			}
		}

		if statusCode >= 200 && statusCode < 300 {
			status2xx++
		} else if statusCode >= 300 && statusCode < 400 {
			status3xx++
		} else if statusCode >= 400 && statusCode < 500 {
			status4xx++
		} else if statusCode >= 500 {
			status5xx++
		}
	}

	summary["total_requests"] = totalRequests
	summary["success_count"] = successCount
	summary["error_count"] = errorCount
	summary["status_2xx"] = status2xx
	summary["status_3xx"] = status3xx
	summary["status_4xx"] = status4xx
	summary["status_5xx"] = status5xx

	if totalRequests > 0 {
		summary["error_rate"] = float64(errorCount) / float64(totalRequests) * 100
	} else {
		summary["error_rate"] = 0.0
	}

	avgDuration := 0
	if len(durations) > 0 {
		avgDuration = int(totalDuration / int64(len(durations)))
		sort.Ints(durations)
		p50Idx := int(float64(len(durations)-1) * 0.50)
		p90Idx := int(float64(len(durations)-1) * 0.90)
		p99Idx := int(float64(len(durations)-1) * 0.99)
		summary["latency_p50_ms"] = durations[p50Idx]
		summary["latency_p90_ms"] = durations[p90Idx]
		summary["latency_p95_ms"] = durations[p90Idx] // backward compat
		summary["latency_p99_ms"] = durations[p99Idx]
	} else {
		summary["latency_p50_ms"] = 0
		summary["latency_p90_ms"] = 0
		summary["latency_p95_ms"] = 0
		summary["latency_p99_ms"] = 0
	}
	summary["avg_duration_ms"] = avgDuration

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

// GetTopEndpoints returns the most used endpoints with method and path from audit_logs
func (r *Registry) GetTopEndpoints(since time.Time, limit int) ([]TopEndpoint, error) {
	rows, err := r.db.Queryx(r.db.Rebind(`
		SELECT a.endpoint_id, e.method, e.path, COUNT(*) as request_count, COALESCE(AVG(a.duration_ms), 0) as avg_duration_ms
		FROM audit_logs a
		LEFT JOIN endpoints e ON e.id = a.endpoint_id
		WHERE a.event_type = 'api_request' AND a.timestamp > ? AND a.endpoint_id IS NOT NULL AND a.endpoint_id != ''
		GROUP BY a.endpoint_id, e.method, e.path
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
		if te.Path == nil || *te.Path == "" {
			fallbackPath := te.EndpointID
			te.Path = &fallbackPath
		}
		if te.Method == nil || *te.Method == "" {
			fallbackMethod := "API"
			te.Method = &fallbackMethod
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
