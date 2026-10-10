package targets

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConnector handles Redis target database operations
type RedisConnector struct{}

// NewRedisConnector creates a new Redis connector
func NewRedisConnector() *RedisConnector {
	return &RedisConnector{}
}

// BuildConnectionString builds a Redis connection string (URI format)
func (c *RedisConnector) BuildConnectionString(host string, port int, dbIndexStr, username, password string, useTLS bool) string {
	if strings.HasPrefix(host, "redis://") || strings.HasPrefix(host, "rediss://") {
		return host
	}

	if port <= 0 {
		port = 6379
	}
	db := 0
	if dbIndexStr != "" {
		if val, err := strconv.Atoi(dbIndexStr); err == nil && val >= 0 {
			db = val
		}
	}

	scheme := "redis"
	if useTLS {
		scheme = "rediss"
	}

	if password != "" {
		encodedPassword := url.QueryEscape(password)
		if username != "" && username != "default" {
			encodedUser := url.QueryEscape(username)
			return fmt.Sprintf("%s://%s:%s@%s:%d/%d", scheme, encodedUser, encodedPassword, host, port, db)
		}
		return fmt.Sprintf("%s://:%s@%s:%d/%d", scheme, encodedPassword, host, port, db)
	}

	return fmt.Sprintf("%s://%s:%d/%d", scheme, host, port, db)
}

// TestConnection tests a Redis connection via PING and returns latency
func (c *RedisConnector) TestConnection(connString string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()

	opts, err := redis.ParseURL(connString)
	if err != nil {
		opts = &redis.Options{
			Addr: connString,
		}
	}

	rdb := redis.NewClient(opts)
	defer func() {
		_ = rdb.Close()
	}()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return 0, fmt.Errorf("Redis connection ping failed: %w", err)
	}

	return time.Since(start), nil
}

// Execute runs basic Redis inspection operations
func (c *RedisConnector) Execute(ctx context.Context, connString, query string, params map[string]interface{}, limit int) ([]Row, error) {
	opts, err := redis.ParseURL(connString)
	if err != nil {
		opts = &redis.Options{Addr: connString}
	}

	rdb := redis.NewClient(opts)
	defer func() {
		_ = rdb.Close()
	}()

	cmd := strings.ToUpper(strings.TrimSpace(query))
	if cmd == "INFO" || cmd == "" {
		info, err := rdb.Info(ctx).Result()
		if err != nil {
			return nil, err
		}
		return []Row{{{Name: "info", Value: info}}}, nil
	}

	parts := strings.Fields(query)
	if len(parts) >= 2 && strings.ToUpper(parts[0]) == "GET" {
		val, err := rdb.Get(ctx, parts[1]).Result()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		return []Row{{{Name: "key", Value: parts[1]}, {Name: "value", Value: val}}}, nil
	}

	return []Row{{{Name: "status", Value: "OK"}}}, nil
}
