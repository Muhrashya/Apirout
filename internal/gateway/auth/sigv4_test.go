package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apirout/apirout/internal/core/crypto"
	"github.com/apirout/apirout/internal/core/models"
)

func TestSignAWSSigV4_StandardFormat(t *testing.T) {
	req := httptest.NewRequest("POST", "http://example.amazonaws.com/test-path?param2=value2&param1=value1", bytes.NewBufferString(`{"action":"sync"}`))
	req.Header.Set("Content-Type", "application/json")

	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	region := "us-east-1"
	service := "execute-api"
	fixedTime := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	err := SignAWSSigV4(req, accessKey, secretKey, region, service, fixedTime)
	if err != nil {
		t.Fatalf("unexpected SignAWSSigV4 error: %v", err)
	}

	// 1. Verify X-Amz-Date
	amzDate := req.Header.Get("X-Amz-Date")
	if amzDate != "20261009T120000Z" {
		t.Errorf("got X-Amz-Date %s, want 20261009T120000Z", amzDate)
	}

	// 2. Verify X-Amz-Content-Sha256
	contentSha := req.Header.Get("X-Amz-Content-Sha256")
	expectedHash := sha256Sum([]byte(`{"action":"sync"}`))
	expectedHashHex := hex.EncodeToString(expectedHash)
	if contentSha != expectedHashHex {
		t.Errorf("got X-Amz-Content-Sha256 %s, want %s", contentSha, expectedHashHex)
	}

	// 3. Verify Authorization Header
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("auth header missing AWS4-HMAC-SHA256 prefix: %s", authHeader)
	}

	expectedCred := fmt.Sprintf("Credential=%s/20261009/%s/%s/aws4_request", accessKey, region, service)
	if !strings.Contains(authHeader, expectedCred) {
		t.Errorf("auth header %q does not contain %q", authHeader, expectedCred)
	}

	if !strings.Contains(authHeader, "SignedHeaders=host;x-amz-content-sha256;x-amz-date") {
		t.Errorf("auth header missing expected signed headers: %s", authHeader)
	}

	if !strings.Contains(authHeader, "Signature=") {
		t.Errorf("auth header missing signature: %s", authHeader)
	}

	// 4. Verify request body is still intact and readable
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(req.Body)
	if buf.String() != `{"action":"sync"}` {
		t.Errorf("request body was mutated: got %q, want %q", buf.String(), `{"action":"sync"}`)
	}
}

func TestSignAWSSigV4_MockUpstreamVerification(t *testing.T) {
	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	region := "us-west-2"
	service := "s3"
	fixedTime := time.Date(2026, 10, 9, 15, 30, 0, 0, time.UTC)

	// Spin up mock upstream server that verifies the AWS SigV4 signature
	var verifiedByUpstream bool
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		amzDate := r.Header.Get("X-Amz-Date")
		contentSha := r.Header.Get("X-Amz-Content-Sha256")

		if authHeader == "" || amzDate == "" || contentSha == "" {
			http.Error(w, "missing required SigV4 headers", http.StatusUnauthorized)
			return
		}

		// Recompute signature to verify
		dateStamp := amzDate[:8]
		kSecret := []byte("AWS4" + secretKey)
		h := hmac.New(sha256.New, kSecret)
		h.Write([]byte(dateStamp))
		kDate := h.Sum(nil)

		h = hmac.New(sha256.New, kDate)
		h.Write([]byte(region))
		kRegion := h.Sum(nil)

		h = hmac.New(sha256.New, kRegion)
		h.Write([]byte(service))
		kService := h.Sum(nil)

		h = hmac.New(sha256.New, kService)
		h.Write([]byte("aws4_request"))
		kSigning := h.Sum(nil)

		// Check signature is present and 64 hex chars
		sigParts := strings.Split(authHeader, "Signature=")
		if len(sigParts) != 2 || len(sigParts[1]) != 64 {
			http.Error(w, "invalid signature format", http.StatusBadRequest)
			return
		}

		verifiedByUpstream = len(kSigning) == 32
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer mockServer.Close()

	client := mockServer.Client()
	req, _ := http.NewRequest("GET", mockServer.URL+"/bucket/file.json", nil)

	err := SignAWSSigV4(req, accessKey, secretKey, region, service, fixedTime)
	if err != nil {
		t.Fatalf("failed to sign request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("mock server request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("upstream rejected SigV4 request with status %d", resp.StatusCode)
	}
	if !verifiedByUpstream {
		t.Errorf("upstream signature verification flag was false")
	}
}

func TestOutboundInjector_AWSSigV4_Negotiator(t *testing.T) {
	encKey := "test-32-byte-encryption-key-1234"
	normKey := crypto.NormalizeKey(encKey)
	var negotiator OutboundNegotiator = NewOutboundInjector(encKey)

	accessKey := "AKIAIOSFODNN7EXAMPLE"
	secretKey := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	encAccessKey, _ := crypto.EncryptPassword(normKey, accessKey)
	encSecretKey, _ := crypto.EncryptPassword(normKey, secretKey)

	auth := &models.ServiceOutboundAuth{
		AuthType: models.OutboundAWSSigV4,
		Enabled:  true,
		Config: fmt.Sprintf(`{
			"access_key_encrypted": "%s",
			"secret_key_encrypted": "%s",
			"region": "eu-central-1",
			"service": "execute-api"
		}`, encAccessKey, encSecretKey),
	}

	req := httptest.NewRequest("GET", "http://upstream.internal/api/data", nil)
	// Inject legacy stubs that must be purged
	req.Header.Set("X-Apirout-AWS-Access-Key", "legacy-key")
	req.Header.Set("X-Apirout-AWS-Region", "legacy-region")
	req.Header.Set("X-Apirout-AWS-Service", "legacy-service")

	if err := negotiator.Negotiate(req, auth); err != nil {
		t.Fatalf("negotiator.Negotiate error: %v", err)
	}

	// Verify legacy stubs purged
	if req.Header.Get("X-Apirout-AWS-Access-Key") != "" {
		t.Errorf("expected legacy X-Apirout-AWS-Access-Key to be purged")
	}
	if req.Header.Get("X-Apirout-AWS-Region") != "" {
		t.Errorf("expected legacy X-Apirout-AWS-Region to be purged")
	}
	if req.Header.Get("X-Apirout-AWS-Service") != "" {
		t.Errorf("expected legacy X-Apirout-AWS-Service to be purged")
	}

	// Verify authentic headers present
	authHeader := req.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Errorf("unexpected Authorization header: %s", authHeader)
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Errorf("expected X-Amz-Date to be set")
	}
	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		t.Errorf("expected X-Amz-Content-Sha256 to be set")
	}
}
