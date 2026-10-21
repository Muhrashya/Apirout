package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Role defines the operation mode for Apirout
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
	// Dedicated Multi-Port Architecture:
	// ProxyPort (:8000)   -> Public API Gateway traffic (/api/*, instant CRUD)
	// AdminPort (:8001)   -> Internal Admin/Management REST API (/api/v1/*)
	// GUIPort (:8002)     -> Web Console UI (Vite SPA)
	// StatusPort (:8100)  -> Health checks & Prometheus metrics (/health, /metrics)
	ProxyPort          string
	AdminPort          string
	GUIPort            string
	StatusPort         string
	AdminAllowedIPs    string // CIDR or single IPs allowed to access Admin API & Console UI
	TrustedProxies     string // Comma-separated CIDR or IP list of trusted reverse proxies/load balancers

	// Compatibility aliases
	ControlPlanePort   string
	DataPlanePort      string
	EnableConsole      bool
	StaticDir          string
	DataDir            string
	LogLevel           string
	JWTSecret          string
	ConsoleJWTSecret   string // Dedicated JWT secret for Control Plane admin users
	GatewayJWTSecret   string // Dedicated JWT secret for Data Plane API consumers
	AdminSecret        string // Hasura-style admin secret for automated/CI access
	JWTTTL             time.Duration
	APITokenTTL        time.Duration
	EncryptionKey      string
	GlobalRateLimitRPM    int
	MaxRequestBodyMB      int // Maximum incoming request body size in megabytes (default: 10)
	AuditLogRetentionDays int // Days to retain audit logs before automated pruning (default: 14)

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

	// Headless Auto-Bootstrap (optional environment provisioning)
	BootstrapAdminEmail    string
	BootstrapAdminPassword string

	// Distributed Rate Limiting
	DistributedRateLimit bool

	CORSAllowedOrigins []string
}

// Load loads unified configuration from environment variables
func Load() (*Config, error) {
	roleRaw := strings.ToLower(getEnvAny("ROLE", "APIROUT_ROLE", "traditional"))
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
		TrustedProxies:     getEnvAny("TRUSTED_PROXIES", "APIROUT_TRUSTED_PROXIES", func() string {
			env := strings.ToLower(getEnvAny("APP_ENV", "APIROUT_ENV", "production"))
			if env == "development" || env == "dev" || env == "test" {
				return "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
			}
			return ""
		}()),
		ControlPlanePort:   adminPort,
		DataPlanePort:      proxyPort,
		EnableConsole:      getEnvBool("ENABLE_CONSOLE", true),
		StaticDir:          getEnvAny("STATIC_DIR", "./web/dist"),
		DataDir:            getEnvAny("DATA_DIR", "./data"),
		LogLevel:           strings.ToLower(getEnvAny("LOG_LEVEL", "APIROUT_LOG_LEVEL", "info")),
		JWTSecret:          getEnvAny("JWT_SECRET", "APIROUT_JWT_SECRET", "default-jwt-secret-key-min-32-characters!"),
		ConsoleJWTSecret:   getEnvAny("CONSOLE_JWT_SECRET", "APIROUT_CONSOLE_JWT_SECRET", ""),
		GatewayJWTSecret:   getEnvAny("GATEWAY_JWT_SECRET", "APIROUT_GATEWAY_JWT_SECRET", ""),
		AdminSecret:        getEnvAny("ADMIN_SECRET", "APIROUT_ADMIN_SECRET", ""),
		EncryptionKey:      getEnvAny("APP_ENCRYPTION_KEY", "APIROUT_ENCRYPTION_KEY", "aes-encryption-key-32-chars-ok!"),
		GlobalRateLimitRPM:    getEnvInt("GLOBAL_RATE_LIMIT_RPM", 600),
		MaxRequestBodyMB:      getEnvInt("MAX_REQUEST_BODY_MB", 10),
		AuditLogRetentionDays: getEnvInt("AUDIT_LOG_RETENTION_DAYS", 14),

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

		BootstrapAdminEmail:    getEnvAny("BOOTSTRAP_ADMIN_EMAIL", "APIROUT_BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapAdminPassword: getEnvAny("BOOTSTRAP_ADMIN_PASSWORD", "APIROUT_BOOTSTRAP_ADMIN_PASSWORD", ""),
		DistributedRateLimit:   getEnvBool("DISTRIBUTED_RATE_LIMIT", false),

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
	if cfg.ConsoleJWTSecret == "" {
		cfg.ConsoleJWTSecret = cfg.JWTSecret
	}
	if cfg.GatewayJWTSecret == "" {
		cfg.GatewayJWTSecret = cfg.JWTSecret
	}

	if cfg.BootstrapAdminEmail != "" && cfg.BootstrapAdminPassword == "" {
		return nil, fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD is required when BOOTSTRAP_ADMIN_EMAIL is set")
	}
	if cfg.BootstrapAdminPassword != "" && cfg.BootstrapAdminEmail == "" {
		return nil, fmt.Errorf("BOOTSTRAP_ADMIN_EMAIL is required when BOOTSTRAP_ADMIN_PASSWORD is set")
	}
	if cfg.BootstrapAdminEmail != "" {
		if !strings.Contains(cfg.BootstrapAdminEmail, "@") {
			return nil, fmt.Errorf("BOOTSTRAP_ADMIN_EMAIL has invalid email format")
		}
		if len(cfg.BootstrapAdminPassword) < 8 {
			return nil, fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be at least 8 characters")
		}
	}

	if cfg.AppEnv == "production" {
		if cfg.BootstrapAdminEmail != "" {
			if cfg.BootstrapAdminPassword == "Password123!" || cfg.BootstrapAdminPassword == "admin" || cfg.BootstrapAdminPassword == "password" {
				return nil, fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must not use default placeholder in production")
			}
		}
		if len(cfg.ConsoleJWTSecret) < 32 || cfg.ConsoleJWTSecret == "default-jwt-secret-key-min-32-characters!" || cfg.ConsoleJWTSecret == "apirout-default-jwt-secret-key-32chars!" || cfg.ConsoleJWTSecret == "super-secret-jwt-key-min-32-characters!" {
			return nil, fmt.Errorf("CONSOLE_JWT_SECRET (or JWT_SECRET) must be at least 32 characters and must not use default placeholder in production")
		}
		if len(cfg.GatewayJWTSecret) < 32 || cfg.GatewayJWTSecret == "default-jwt-secret-key-min-32-characters!" || cfg.GatewayJWTSecret == "apirout-default-jwt-secret-key-32chars!" || cfg.GatewayJWTSecret == "super-secret-jwt-key-min-32-characters!" {
			return nil, fmt.Errorf("GATEWAY_JWT_SECRET (or JWT_SECRET) must be at least 32 characters and must not use default placeholder in production")
		}
		if len(cfg.EncryptionKey) < 32 || cfg.EncryptionKey == "aes-encryption-key-32-chars-ok!" || cfg.EncryptionKey == "aes-encryption-key-32-characters!" || cfg.EncryptionKey == "apirout-local-dev-aes-encryption-key-32!" {
			return nil, fmt.Errorf("APP_ENCRYPTION_KEY must be at least 32 characters and must not use default placeholder in production")
		}
		if cfg.AdminSecret == "" {
			return nil, fmt.Errorf("ADMIN_SECRET must be set in production to secure administrative APIs")
		}
		if cfg.AdminSecret == "MasterSecretKey2026!" || cfg.AdminSecret == "APIROUT_ADMIN_SECRET" {
			return nil, fmt.Errorf("ADMIN_SECRET must not use default placeholder in production")
		}
		if len(cfg.AdminSecret) < 24 {
			return nil, fmt.Errorf("ADMIN_SECRET must be at least 24 characters in production")
		}
		if cfg.DBType == "postgres" || cfg.DBType == "postgresql" || cfg.DBType == "mysql" {
			if cfg.DBPassword == "" {
				return nil, fmt.Errorf("DB_PASSWORD must not be empty in production for database type %s", cfg.DBType)
			}
			if cfg.DBPassword == "Password123!" || cfg.DBPassword == "ProductionSecurePassword123!" || cfg.DBPassword == "postgres" || cfg.DBPassword == "admin" {
				return nil, fmt.Errorf("DB_PASSWORD must not use default placeholder in production")
			}
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
