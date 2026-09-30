package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Role defines the operation mode for Apirout (Kong Gateway & Hasura style)
type Role string

const (
	// RoleTraditional runs both Control Plane and Data Plane in one process (default)
	RoleTraditional Role = "traditional"
	// RoleControlPlane runs Admin Console UI, Management APIs, GitOps, and DB Migrations
	RoleControlPlane Role = "control_plane"
	// RoleDataPlane runs the stateless, high-throughput API Gateway proxy
	RoleDataPlane Role = "data_plane"
)

// Config represents runtime configuration for Apirout
type Config struct {
	Role               Role
	AppEnv             string
	// Kong-style Dedicated Multi-Port Architecture:
	// ProxyPort (:8000)   -> Public API Gateway traffic (/api/*, instant CRUD)
	// AdminPort (:8001)   -> Internal Admin/Management REST API (/api/v1/*)
	// GUIPort (:8002)     -> Web Console UI (Vite SPA)
	// StatusPort (:8100)  -> Health checks & Prometheus metrics (/health, /metrics)
	ProxyPort          string
	AdminPort          string
	GUIPort            string
	StatusPort         string
	AdminAllowedIPs    string // CIDR or single IPs allowed to access Admin API & Console UI

	// Compatibility aliases
	ControlPlanePort   string
	DataPlanePort      string
	EnableConsole      bool
	StaticDir          string
	DataDir            string
	LogLevel           string
	JWTSecret          string
	AdminSecret        string // Hasura-style admin secret for automated/CI access
	JWTTTL             time.Duration
	APITokenTTL        time.Duration
	AdminEmail         string
	AdminPassword      string
	EncryptionKey      string
	GlobalRateLimitRPM int

	// Registry Database (PostgreSQL / SQLite)
	DBType            string
	DBPath            string
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	// Redis for Pub/Sub invalidation and caching
	RedisURL      string
	RedisHost     string
	RedisPort     string
	RedisPassword string

	CORSAllowedOrigins []string
}

// Load loads unified configuration from environment variables
func Load() (*Config, error) {
	roleRaw := strings.ToLower(getEnvAny("ROLE", "APIROUT_ROLE", "KONG_ROLE", "traditional"))
	var role Role
	switch roleRaw {
	case "control_plane", "control-plane", "cp", "console":
		role = RoleControlPlane
	case "data_plane", "data-plane", "dp", "gateway":
		role = RoleDataPlane
	case "traditional", "all", "standalone":
		role = RoleTraditional
	default:
		role = RoleTraditional
	}

	proxyPort := getEnvAny("PROXY_PORT", "APIROUT_PROXY_LISTEN", "APIROUT_GATEWAY_PORT", "8000")
	adminPort := getEnvAny("ADMIN_PORT", "APIROUT_ADMIN_LISTEN", "APIROUT_ADMIN_PORT", "8001")
	guiPort := getEnvAny("GUI_PORT", "APIROUT_GUI_LISTEN", "APIROUT_CONSOLE_PORT", "8002")
	statusPort := getEnvAny("STATUS_PORT", "APIROUT_STATUS_LISTEN", "APIROUT_STATUS_PORT", "8100")

	if legacyPort := os.Getenv("PORT"); legacyPort != "" {
		if role == RoleControlPlane {
			adminPort = legacyPort
		} else if role == RoleDataPlane {
			proxyPort = legacyPort
		}
	}

	cfg := &Config{
		Role:               role,
		AppEnv:             getEnvAny("APP_ENV", "APIROUT_ENV", "production"),
		ProxyPort:          proxyPort,
		AdminPort:          adminPort,
		GUIPort:            guiPort,
		StatusPort:         statusPort,
		AdminAllowedIPs:    getEnvAny("ADMIN_ALLOWED_IPS", "APIROUT_ADMIN_ALLOWED_IPS", "*"),
		ControlPlanePort:   adminPort,
		DataPlanePort:      proxyPort,
		EnableConsole:      getEnvBool("ENABLE_CONSOLE", true),
		StaticDir:          getEnvAny("STATIC_DIR", "./web/dist"),
		DataDir:            getEnvAny("DATA_DIR", "./data"),
		LogLevel:           strings.ToLower(getEnvAny("LOG_LEVEL", "APIROUT_LOG_LEVEL", "info")),
		JWTSecret:          getEnvAny("JWT_SECRET", "APIROUT_JWT_SECRET", "default-jwt-secret-key-min-32-characters!"),
		AdminSecret:        getEnvAny("ADMIN_SECRET", "APIROUT_ADMIN_SECRET", ""),
		AdminEmail:         getEnvAny("ADMIN_EMAIL", "APIROUT_ADMIN_EMAIL", ""),
		AdminPassword:      getEnvAny("ADMIN_PASSWORD", "APIROUT_ADMIN_PASSWORD", ""),
		EncryptionKey:      getEnvAny("APP_ENCRYPTION_KEY", "APIROUT_ENCRYPTION_KEY", "aes-encryption-key-32-chars-ok!"),
		GlobalRateLimitRPM: getEnvInt("GLOBAL_RATE_LIMIT_RPM", 600),

		DBType:            strings.ToLower(getEnvAny("DB_TYPE", "APIROUT_DB_TYPE", "postgres")),
		DBPath:            getEnvAny("DB_PATH", "APIROUT_DB_PATH", ""),
		DBHost:            getEnvAny("DB_HOST", "APIROUT_DB_HOST", "localhost"),
		DBPort:            getEnvAny("DB_PORT", "APIROUT_DB_PORT", "5432"),
		DBName:            getEnvAny("DB_NAME", "POSTGRES_DB", "APIROUT_DB_NAME", "main_db"),
		DBUser:            getEnvAny("DB_USER", "POSTGRES_USER", "APIROUT_DB_USER", "postgres"),
		DBPassword:        getEnvAny("DB_PASSWORD", "POSTGRES_PASSWORD", "APIROUT_DB_PASSWORD", ""),
		DBSSLMode:         getEnvAny("DB_SSLMODE", "APIROUT_DB_SSLMODE", "disable"),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		DBConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute),

		RedisURL:      getEnvAny("REDIS_URL", "APIROUT_REDIS_URL", ""),
		RedisHost:     getEnvAny("REDIS_HOST", "APIROUT_REDIS_HOST", "localhost"),
		RedisPort:     getEnvAny("REDIS_PORT", "APIROUT_REDIS_PORT", "6379"),
		RedisPassword: getEnvAny("REDIS_PASSWORD", "APIROUT_REDIS_PASSWORD", ""),

		CORSAllowedOrigins: splitList(getEnvAny("CORS_ALLOWED_ORIGINS", "*")),
	}

	// Support Hasura-style DATABASE_URL parsing
	if dbURL := getEnvAny("APIROUT_DATABASE_URL", "DATABASE_URL", ""); dbURL != "" {
		parseDatabaseURL(dbURL, cfg)
	}

	ttlMin := getEnvInt("JWT_TTL_MINUTES", 1440)
	cfg.JWTTTL = time.Duration(ttlMin) * time.Minute

	apiTokenTTLMin := getEnvInt("API_TOKEN_TTL_MINUTES", 43200)
	cfg.APITokenTTL = time.Duration(apiTokenTTLMin) * time.Minute

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "apirout-default-jwt-secret-key-32chars!"
	}

	if cfg.AppEnv == "production" {
		if len(cfg.JWTSecret) < 32 || cfg.JWTSecret == "default-jwt-secret-key-min-32-characters!" || cfg.JWTSecret == "apirout-default-jwt-secret-key-32chars!" {
			return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters and must not use default placeholder in production")
		}
		if len(cfg.EncryptionKey) < 32 || cfg.EncryptionKey == "aes-encryption-key-32-chars-ok!" {
			return nil, fmt.Errorf("APP_ENCRYPTION_KEY must be at least 32 characters and must not use default placeholder in production")
		}
	}

	return cfg, nil
}

func getEnvAny(keys ...string) string {
	for i, key := range keys {
		if i == len(keys)-1 && !strings.Contains(key, "_") && !strings.Contains(key, ".") {
			// might be default value
		}
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	if len(keys) > 0 {
		last := keys[len(keys)-1]
		if !strings.Contains(last, "_") || strings.Contains(last, " ") || last == "" || last == "postgres" || last == "localhost" || last == "apirout" || last == "info" || last == "traditional" || last == "all" || last == "disable" || last == "*" || last == "production" || last == "8081" || last == "8080" || last == "5432" || last == "6379" || strings.Contains(last, "@") || strings.Contains(last, "!") || strings.Contains(last, "./") {
			return last
		}
	}
	return ""
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func parseDatabaseURL(dbURL string, cfg *Config) {
	// Format: postgres://user:password@host:port/dbname?sslmode=disable
	if !strings.HasPrefix(dbURL, "postgres://") && !strings.HasPrefix(dbURL, "postgresql://") {
		return
	}
	cfg.DBType = "postgres"
	trimmed := strings.TrimPrefix(strings.TrimPrefix(dbURL, "postgres://"), "postgresql://")
	
	// Split user:pass and rest
	parts := strings.SplitN(trimmed, "@", 2)
	if len(parts) == 2 {
		userPass := strings.SplitN(parts[0], ":", 2)
		cfg.DBUser = userPass[0]
		if len(userPass) == 2 {
			cfg.DBPassword = userPass[1]
		}
		trimmed = parts[1]
	}
	
	// Split host:port and dbname?sslmode
	hostDb := strings.SplitN(trimmed, "/", 2)
	if len(hostDb) > 0 {
		hostPort := strings.SplitN(hostDb[0], ":", 2)
		cfg.DBHost = hostPort[0]
		if len(hostPort) == 2 {
			cfg.DBPort = hostPort[1]
		}
	}
	if len(hostDb) == 2 {
		dbAndQuery := strings.SplitN(hostDb[1], "?", 2)
		cfg.DBName = dbAndQuery[0]
		if len(dbAndQuery) == 2 {
			queries := strings.Split(dbAndQuery[1], "&")
			for _, q := range queries {
				kv := strings.SplitN(q, "=", 2)
				if len(kv) == 2 && kv[0] == "sslmode" {
					cfg.DBSSLMode = kv[1]
				}
			}
		}
	}
}

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
		if s, err := strconv.Atoi(val); err == nil {
			return time.Duration(s) * time.Second
		}
	}
	return defaultVal
}

func splitList(val string) []string {
	if val == "" {
		return []string{"*"}
	}
	items := strings.Split(val, ",")
	res := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it != "" {
			res = append(res, it)
		}
	}
	if len(res) == 0 {
		return []string{"*"}
	}
	return res
}
