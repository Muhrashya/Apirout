package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// SignAWSSigV4 computes and attaches authentic AWS Signature Version 4 headers to an HTTP request
func SignAWSSigV4(req *http.Request, accessKey, secretKey, region, service string, signTime time.Time) error {
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("aws-sigv4: access key and secret key are required")
	}
	if region == "" {
		region = "us-east-1"
	}
	if service == "" {
		service = "execute-api"
	}

	amzDate := signTime.UTC().Format("20060102T150405Z")
	dateStamp := signTime.UTC().Format("20060102")

	// 1. Calculate Payload Hash (SHA256 hex)
	payloadHash, err := getPayloadHash(req)
	if err != nil {
		return fmt.Errorf("aws-sigv4: failed to read payload: %w", err)
	}

	// Set required headers for SigV4
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	// Determine host header
	host := req.Host
	if host == "" && req.URL != nil {
		host = req.URL.Host
	}
	if host == "" {
		host = "localhost"
	}

	// 2. Build Canonical Request
	// Method
	method := req.Method
	if method == "" {
		method = "GET"
	}

	// Canonical URI
	canonicalURI := "/"
	if req.URL != nil && req.URL.Path != "" {
		canonicalURI = uriEncode(req.URL.Path, false)
		if !strings.HasPrefix(canonicalURI, "/") {
			canonicalURI = "/" + canonicalURI
		}
	}

	// Canonical Query String
	canonicalQuery := ""
	if req.URL != nil && req.URL.RawQuery != "" {
		canonicalQuery = buildCanonicalQuery(req.URL.Query())
	}

	// Canonical Headers and Signed Headers
	// Headers to sign: host, x-amz-content-sha256, x-amz-date (sorted alphabetically)
	headersToSign := map[string]string{
		"host":                 strings.TrimSpace(host),
		"x-amz-content-sha256": strings.TrimSpace(payloadHash),
		"x-amz-date":           strings.TrimSpace(amzDate),
	}

	signedHeaderKeys := make([]string, 0, len(headersToSign))
	for k := range headersToSign {
		signedHeaderKeys = append(signedHeaderKeys, k)
	}
	sort.Strings(signedHeaderKeys)

	var canonicalHeaders strings.Builder
	for _, k := range signedHeaderKeys {
		canonicalHeaders.WriteString(fmt.Sprintf("%s:%s\n", k, headersToSign[k]))
	}

	signedHeaders := strings.Join(signedHeaderKeys, ";")

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	)

	// 3. Create String to Sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	hashedCanonicalRequest := hex.EncodeToString(sha256Sum([]byte(canonicalRequest)))

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hashedCanonicalRequest,
	)

	// 4. Calculate Signature
	signingKey := getAWSSignatureKey(secretKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// 5. Build Authorization Header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)

	req.Header.Set("Authorization", authHeader)
	return nil
}

func getPayloadHash(req *http.Request) (string, error) {
	if req.Body == nil {
		emptyHash := sha256Sum([]byte(""))
		return hex.EncodeToString(emptyHash), nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	_ = req.Body.Close()
	// Restore body for subsequent proxy forwarding
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	hash := sha256Sum(bodyBytes)
	return hex.EncodeToString(hash), nil
}

func buildCanonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		vals := values[k]
		sort.Strings(vals)
		escapedKey := uriEncode(k, true)
		if len(vals) == 0 {
			parts = append(parts, escapedKey+"=")
		} else {
			for _, v := range vals {
				parts = append(parts, escapedKey+"="+uriEncode(v, true))
			}
		}
	}
	return strings.Join(parts, "&")
}

func uriEncode(s string, encodeSlash bool) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			buf.WriteByte(c)
		} else if c == '/' && !encodeSlash {
			buf.WriteByte('/')
		} else {
			buf.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return buf.String()
}

func sha256Sum(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getAWSSignatureKey(secretKey, dateStamp, region, service string) []byte {
	kSecret := []byte("AWS4" + secretKey)
	kDate := hmacSHA256(kSecret, []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
