package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	consoleAuth "github.com/apirout/apirout/internal/console/auth"
	consoleExecutor "github.com/apirout/apirout/internal/console/executor"
	consoleRegistry "github.com/apirout/apirout/internal/console/registry"
	"github.com/apirout/apirout/internal/console/server"
	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/egress"
	"github.com/apirout/apirout/internal/core/targets"
	gatewayAuth "github.com/apirout/apirout/internal/gateway/auth"
	gatewayExecutor "github.com/apirout/apirout/internal/gateway/executor"
	"github.com/golang-jwt/jwt/v5"
)

// 1. E2E: SSRF Shield rejects loopback, link-local metadata, and private network destinations
func TestE2E_SecurityHardening_SSRFProtection(t *testing.T) {
	blockedHosts := []string{
		"127.0.0.1",
		"localhost",
		"127.0.0.1:8080",
		"localhost:5432",
		"169.254.169.254",
		"169.254.169.254:80",
		"metadata.google.internal",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
	}

	for _, host := range blockedHosts {
		if err := egress.ValidateSafeHost(host); err == nil {
			t.Errorf("expected SSRF block for host %q, but got nil", host)
		}
	}

	// URL validator checks
	blockedURLs := []string{
		"http://127.0.0.1/admin",
		"http://localhost:3000/api",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://10.0.0.5/internal/status",
	}

	for _, rawURL := range blockedURLs {
		if err := egress.ValidateSafeURL(rawURL); err == nil {
			t.Errorf("expected SSRF block for URL %q, but got nil", rawURL)
		}
	}

	// OAuth2CCManager SSRF boundary check
	ccMgr := gatewayAuth.NewOAuth2CCManager()
	for _, rawURL := range blockedURLs {
		_, err := ccMgr.GetToken(rawURL, "client-id", "client-secret", "")
		if err == nil {
			t.Errorf("expected OAuth2CCManager to reject SSRF token URL %q, but succeeded", rawURL)
		}
	}

	// ConnectionHandler.TestRaw rejects probing loopback and metadata
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "ssrf_test.db")
	reg, err := consoleRegistry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	connHandler := server.NewConnectionHandler(reg, "test-encryption-key-32-chars-long!")

	for _, badHost := range []string{"127.0.0.1", "localhost", "169.254.169.254"} {
		payload := map[string]interface{}{
			"db_type":  "postgres",
			"host":     badHost,
			"port":     5432,
			"dbname":   "test",
			"username": "postgres",
			"password": "password",
		}
		bodyBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/connections/test-raw", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
			UserID: "admin-id",
			Role:   consoleAuth.RoleSuperAdmin,
		})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		connHandler.TestRaw(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected HTTP 400 for probing SSRF host %q, got: %d", badHost, rec.Code)
		}
		var resp server.TestConnectionResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err == nil {
			if resp.Success {
				t.Errorf("expected Success=false for SSRF host %q", badHost)
			}
			if resp.Error == nil || !strings.Contains(*resp.Error, "Security violation") {
				t.Errorf("expected Security violation in error message for %q, got: %v", badHost, resp.Error)
			}
		}
	}
}

// 2. E2E: SQL validator blocks file read/write, sleep delays, and benchmarks
func TestE2E_SecurityHardening_SQLValidator(t *testing.T) {
	dangerousQueries := []string{
		"SELECT * FROM users INTO OUTFILE '/var/www/html/shell.php'",
		"SELECT * FROM users INTO DUMPFILE '/tmp/dump.bin'",
		"SELECT LOAD_FILE('/etc/passwd')",
		"SELECT pg_read_file('/etc/passwd')",
		"SELECT pg_read_binary_file('/etc/shadow')",
		"SELECT lo_export(1001, '/tmp/target')",
		"SELECT lo_import('/tmp/source')",
		"SELECT * FROM users WHERE id = 1 AND SLEEP(5)",
		"SELECT pg_sleep(10)",
		"SELECT BENCHMARK(5000000, MD5('test'))",
		"SELECT xp_cmdshell('whoami')",
	}

	for _, q := range dangerousQueries {
		if err := consoleExecutor.ValidateSQL(q); err == nil {
			t.Errorf("console executor allowed dangerous SQL: %s", q)
		}
		if err := gatewayExecutor.ValidateSQL(q); err == nil {
			t.Errorf("gateway executor allowed dangerous SQL: %s", q)
		}
	}

	// Legitimate queries pass cleanly
	safeQueries := []string{
		"SELECT id, name, email FROM accounts WHERE status = 'active'",
		"SELECT count(1) FROM audit_logs WHERE created_at >= NOW()",
		"WITH active_users AS (SELECT id FROM users WHERE active = true) SELECT * FROM active_users",
		"SELECT first_name, last_name, dept FROM employees ORDER BY dept, last_name",
	}

	for _, q := range safeQueries {
		if err := consoleExecutor.ValidateSQL(q); err != nil {
			t.Errorf("console executor rejected safe query: %s, error: %v", q, err)
		}
		if err := gatewayExecutor.ValidateSQL(q); err != nil {
			t.Errorf("gateway executor rejected safe query: %s, error: %v", q, err)
		}
	}
}

// 3. E2E: Cryptographic separation prevents Control Plane admin tokens from accessing Data Plane
func TestE2E_SecurityHardening_CrossPlaneTokenSeparation(t *testing.T) {
	sharedKey := []byte("cryptographic-test-secret-key-32")

	// Control Plane admin service
	authService := consoleAuth.NewService(string(sharedKey), nil, 60, "admin-secret")
	adminToken, _, err := authService.GenerateToken(&consoleRegistry.User{
		ID:    "superadmin-123",
		Email: "superadmin@apirout.io",
		Role:  consoleAuth.RoleSuperAdmin,
	})
	if err != nil {
		t.Fatalf("failed to generate console admin token: %v", err)
	}

	// Gateway Data Plane JWT auth plugin
	plugin := gatewayAuth.NewEngine(nil, string(sharedKey))
	req := httptest.NewRequest("GET", "/api/v1/customers", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	// Attempt authenticating with admin token via gateway engine
	res, err := plugin.Authenticate(req, nil, nil)
	if err == nil {
		t.Fatalf("expected gateway to reject control plane admin token, but succeeded: %+v", res)
	}

	// Direct check on JWTAuthPlugin: ensure typ: admin is rejected
	jwtPlugin := gatewayAuth.NewJWTAuthPlugin(sharedKey)
	claims := jwt.MapClaims{
		"sub":     "superadmin-123",
		"user_id": "superadmin-123",
		"typ":     "admin",
		"role":    "SUPER_ADMIN",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(sharedKey)

	req2 := httptest.NewRequest("GET", "/api/v1/customers", nil)
	req2.Header.Set("Authorization", "Bearer "+tokenStr)

	_, err = jwtPlugin.Authenticate(req2, map[string]interface{}{}, nil)
	if err == nil {
		t.Fatalf("expected JWTAuthPlugin to reject admin claim (typ: admin), but it passed")
	}
	if !strings.Contains(err.Error(), "cross-plane token rejected") {
		t.Errorf("expected cross-plane token rejection error, got: %v", err)
	}

	// Consumer token with client sub succeeds
	consumerClaims := jwt.MapClaims{
		"sub": "api-consumer-abc",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	consumerToken := jwt.NewWithClaims(jwt.SigningMethodHS256, consumerClaims)
	consumerTokenStr, _ := consumerToken.SignedString(sharedKey)

	reqConsumer := httptest.NewRequest("GET", "/api/v1/customers", nil)
	reqConsumer.Header.Set("Authorization", "Bearer "+consumerTokenStr)

	authRes, err := jwtPlugin.Authenticate(reqConsumer, map[string]interface{}{}, nil)
	if err != nil {
		t.Fatalf("expected valid consumer token to be authenticated, got error: %v", err)
	}
	if authRes.ConsumerID != "api-consumer-abc" {
		t.Errorf("expected consumer ID 'api-consumer-abc', got: %s", authRes.ConsumerID)
	}
	_ = res
}

// 4. E2E: RBAC restricts user enumeration exclusively to SUPER_ADMIN
func TestE2E_SecurityHardening_UserEnumerationRBAC(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "rbac_test.db")
	reg, err := consoleRegistry.NewWithSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed to init test registry: %v", err)
	}
	defer reg.Close()

	if err := reg.Migrate(); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	userHandler := server.NewUserHandler(reg)

	// 1. DEVELOPER gets 403
	reqDev := httptest.NewRequest("GET", "/api/v1/users", nil)
	ctxDev := context.WithValue(reqDev.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
		UserID: "dev-user",
		Role:   consoleAuth.RoleDeveloper,
	})
	recDev := httptest.NewRecorder()
	userHandler.List(recDev, reqDev.WithContext(ctxDev))
	if recDev.Code != http.StatusForbidden {
		t.Errorf("expected HTTP 403 Forbidden for DEVELOPER, got: %d", recDev.Code)
	}

	// 2. MAINTAINER gets 403
	reqMaint := httptest.NewRequest("GET", "/api/v1/users", nil)
	ctxMaint := context.WithValue(reqMaint.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
		UserID: "maint-user",
		Role:   consoleAuth.RoleMaintainer,
	})
	recMaint := httptest.NewRecorder()
	userHandler.List(recMaint, reqMaint.WithContext(ctxMaint))
	if recMaint.Code != http.StatusForbidden {
		t.Errorf("expected HTTP 403 Forbidden for MAINTAINER, got: %d", recMaint.Code)
	}

	// 3. Anonymous gets 403
	reqAnon := httptest.NewRequest("GET", "/api/v1/users", nil)
	recAnon := httptest.NewRecorder()
	userHandler.List(recAnon, reqAnon)
	if recAnon.Code != http.StatusForbidden {
		t.Errorf("expected HTTP 403 Forbidden for Anonymous, got: %d", recAnon.Code)
	}

	// 4. SUPER_ADMIN gets 200
	reqSuper := httptest.NewRequest("GET", "/api/v1/users", nil)
	ctxSuper := context.WithValue(reqSuper.Context(), consoleAuth.UserContextKey, &consoleAuth.Claims{
		UserID: "super-user",
		Role:   consoleAuth.RoleSuperAdmin,
	})
	recSuper := httptest.NewRecorder()
	userHandler.List(recSuper, reqSuper.WithContext(ctxSuper))
	if recSuper.Code != http.StatusOK {
		t.Errorf("expected HTTP 200 OK for SUPER_ADMIN, got: %d", recSuper.Code)
	}
}

// 5. E2E: Authenticated AES-256-GCM encryption, tampering detection, and secret redaction
func TestE2E_SecurityHardening_EncryptionAndRedaction(t *testing.T) {
	key := crypto.NormalizeKey("prod-master-encryption-key-32b!")
	password := "SecretP@ssword2026!"

	// Encrypt
	ciphertext, err := crypto.EncryptPassword(key, password)
	if err != nil {
		t.Fatalf("failed to encrypt password: %v", err)
	}
	if !strings.HasPrefix(ciphertext, "gcm:") {
		t.Errorf("expected gcm: prefix on authenticated ciphertext, got: %s", ciphertext)
	}

	// Decrypt valid
	decrypted, err := crypto.DecryptPassword(key, ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt valid ciphertext: %v", err)
	}
	if decrypted != password {
		t.Errorf("expected %q, got %q", password, decrypted)
	}

	// Tampered ciphertext fails
	rawBytes := []byte(ciphertext)
	if rawBytes[len(rawBytes)-2] == 'A' {
		rawBytes[len(rawBytes)-2] = 'Z'
	} else {
		rawBytes[len(rawBytes)-2] = 'A'
	}
	tampered := string(rawBytes)

	_, err = crypto.DecryptPassword(key, tampered)
	if err == nil {
		t.Fatalf("expected AEAD error when decrypting tampered ciphertext, but succeeded")
	}

	// Inbound auth plugin encryption & redaction
	pluginConfig := `{
		"secret": "my-hmac-secret-key",
		"password": "my-basic-auth-password",
		"allowed_origins": ["https://app.apirout.io"]
	}`

	encConfig, err := server.EncryptInboundAuthConfig(pluginConfig, key)
	if err != nil {
		t.Fatalf("EncryptInboundAuthConfig failed: %v", err)
	}
	if strings.Contains(encConfig, "my-hmac-secret-key") || strings.Contains(encConfig, "my-basic-auth-password") {
		t.Errorf("encrypted inbound config leaked plaintext secrets: %s", encConfig)
	}

	redactedConfig := server.RedactInboundAuthConfig(encConfig)
	var parsedRedacted map[string]interface{}
	if err := json.Unmarshal([]byte(redactedConfig), &parsedRedacted); err != nil {
		t.Fatalf("failed to parse redacted config: %v", err)
	}

	if parsedRedacted["secret_configured"] != true || parsedRedacted["password_configured"] != true {
		t.Errorf("expected boolean configured markers in redacted config: %+v", parsedRedacted)
	}
	if _, isStr := parsedRedacted["secret_encrypted"].(string); isStr {
		t.Errorf("redacted config leaked secret_encrypted string ciphertext")
	}
}

// 6. E2E: Safe connection string construction URL-encodes special characters
func TestE2E_SecurityHardening_SafeDSNConstruction(t *testing.T) {
	user := "svc_user@domain:sub"
	pass := "p@$$w0rd!#%&/?="

	dialects := []string{"postgres", "mysql", "mssql", "mongodb", "redis"}
	for _, dialect := range dialects {
		dsn, err := targets.BuildConnectionString(dialect, "db-host.internal", 5432, "app_db", user, pass, "disable")
		if err != nil {
			t.Fatalf("failed to build %s connection string: %v", dialect, err)
		}

		if strings.Contains(dsn, user) {
			t.Errorf("%s DSN contains unencoded username: %s", dialect, dsn)
		}
		if strings.Contains(dsn, pass) {
			t.Errorf("%s DSN contains unencoded password: %s", dialect, dsn)
		}
	}
}
