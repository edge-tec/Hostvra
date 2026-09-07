package backup

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// S3Config holds credentials and endpoint info for S3-compatible cloud storage.
type S3Config struct {
	Endpoint  string // e.g. "s3.us-east-1.amazonaws.com" or "https://<account>.r2.cloudflarestorage.com"
	Region    string // e.g. "us-east-1", "auto" for R2
	Bucket    string // Target bucket name
	AccessKey string // S3 Access Key ID
	SecretKey string // S3 Secret Access Key
	Prefix    string // Optional prefix/path in bucket
	UseSSL    bool
}

// S3Client handles S3-compatible object operations using pure Go standard library with AWS SigV4.
type S3Client struct {
	config     S3Config
	httpClient *http.Client
}

// NewS3Client creates a new S3Client instance.
func NewS3Client(cfg S3Config) *S3Client {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	return &S3Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// getEndpointURL resolves the host and URL scheme.
func (c *S3Client) getEndpointURL() (*url.URL, error) {
	raw := strings.TrimSpace(c.config.Endpoint)
	if raw == "" {
		raw = "s3." + c.config.Region + ".amazonaws.com"
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if c.config.UseSSL {
			raw = "https://" + raw
		} else {
			raw = "http://" + raw
		}
	}
	return url.Parse(raw)
}

// signV4 generates the AWS SigV4 Authorization header and headers map.
func (c *S3Client) signV4(method, canonicalURI, query string, headers map[string]string, payloadHash string, t time.Time) {
	dateStamp := t.UTC().Format("20060102")
	amzDate := t.UTC().Format("20060102T150405Z")

	headers["x-amz-date"] = amzDate
	headers["x-amz-content-sha256"] = payloadHash

	// Canonical headers
	var signedHeadersSlice []string
	canonicalHeaders := ""
	for k := range headers {
		signedHeadersSlice = append(signedHeadersSlice, strings.ToLower(k))
	}
	// Sort headers
	for i := 0; i < len(signedHeadersSlice); i++ {
		for j := i + 1; j < len(signedHeadersSlice); j++ {
			if signedHeadersSlice[i] > signedHeadersSlice[j] {
				signedHeadersSlice[i], signedHeadersSlice[j] = signedHeadersSlice[j], signedHeadersSlice[i]
			}
		}
	}

	for _, k := range signedHeadersSlice {
		val := strings.TrimSpace(headers[k])
		canonicalHeaders += fmt.Sprintf("%s:%s\n", k, val)
	}
	signedHeaders := strings.Join(signedHeadersSlice, ";")

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method,
		canonicalURI,
		query,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	reqHash := sha256Hex([]byte(canonicalRequest))

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.config.Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate,
		credentialScope,
		reqHash,
	)

	signingKey := getSignatureKey(c.config.SecretKey, dateStamp, c.config.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.config.AccessKey,
		credentialScope,
		signedHeaders,
		signature,
	)
	headers["authorization"] = authHeader
}

// PutObject uploads data to the specified key in the S3 bucket.
func (c *S3Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	endpoint, err := c.getEndpointURL()
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	if contentType == "" {
		contentType = "application/gzip"
	}

	// Clean key
	key = strings.TrimPrefix(key, "/")
	if c.config.Prefix != "" {
		prefix := strings.Trim(c.config.Prefix, "/")
		key = prefix + "/" + key
	}

	// Path-style URL: /bucket/key
	canonicalURI := "/" + c.config.Bucket + "/" + key
	targetURL := fmt.Sprintf("%s://%s%s", endpoint.Scheme, endpoint.Host, canonicalURI)

	t := time.Now().UTC()
	payloadHash := sha256Hex(data)

	headers := map[string]string{
		"host":         endpoint.Host,
		"content-type": contentType,
	}

	c.signV4("PUT", canonicalURI, "", headers, payloadHash, t)

	req, err := http.NewRequestWithContext(ctx, "PUT", targetURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload to s3 failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 upload returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteObject removes an object from the S3 bucket.
func (c *S3Client) DeleteObject(ctx context.Context, key string) error {
	endpoint, err := c.getEndpointURL()
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	key = strings.TrimPrefix(key, "/")
	if c.config.Prefix != "" {
		prefix := strings.Trim(c.config.Prefix, "/")
		key = prefix + "/" + key
	}

	canonicalURI := "/" + c.config.Bucket + "/" + key
	targetURL := fmt.Sprintf("%s://%s%s", endpoint.Scheme, endpoint.Host, canonicalURI)

	t := time.Now().UTC()
	payloadHash := sha256Hex([]byte(""))

	headers := map[string]string{
		"host": endpoint.Host,
	}

	c.signV4("DELETE", canonicalURI, "", headers, payloadHash, t)

	req, err := http.NewRequestWithContext(ctx, "DELETE", targetURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestConnection validates bucket connectivity and credentials using a HEAD or GET bucket probe.
func (c *S3Client) TestConnection(ctx context.Context) error {
	endpoint, err := c.getEndpointURL()
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	canonicalURI := "/" + c.config.Bucket
	targetURL := fmt.Sprintf("%s://%s%s?max-keys=1", endpoint.Scheme, endpoint.Host, canonicalURI)

	t := time.Now().UTC()
	payloadHash := sha256Hex([]byte(""))

	headers := map[string]string{
		"host": endpoint.Host,
	}

	c.signV4("GET", canonicalURI, "max-keys=1", headers, payloadHash, t)

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection probe failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 test returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper hashing and HMAC utilities for AWS SigV4
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(secretKey, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
