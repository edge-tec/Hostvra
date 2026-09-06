package backup

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type S3Config struct {
	Endpoint        string `json:"endpoint"` // e.g. "s3.us-east-1.amazonaws.com" or "https://<account>.r2.cloudflarestorage.com"
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl"`
}

type S3Client struct {
	config     S3Config
	httpClient *http.Client
}

func NewS3Client(config S3Config) *S3Client {
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	return &S3Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // Allow large backup file transfers
		},
	}
}

// UploadFile streams a backup archive to an S3-compatible bucket using AWS Signature v4
func (c *S3Client) UploadFile(localFilePath, s3Key string) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local backup file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	// Hash payload
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	payloadSHA256 := hex.EncodeToString(hash.Sum(nil))

	// Seek back to start
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	endpoint := c.config.Endpoint
	scheme := "https"
	if !c.config.UseSSL && !strings.HasPrefix(endpoint, "https://") {
		scheme = "http"
	}
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	var reqURL string
	var hostHeader string
	if strings.Contains(endpoint, "amazonaws.com") {
		reqURL = fmt.Sprintf("%s://%s.%s/%s", scheme, c.config.Bucket, endpoint, url.PathEscape(s3Key))
		hostHeader = fmt.Sprintf("%s.%s", c.config.Bucket, endpoint)
	} else {
		reqURL = fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, c.config.Bucket, url.PathEscape(s3Key))
		hostHeader = endpoint
	}

	req, err := http.NewRequest(http.MethodPut, reqURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req.Header.Set("Host", hostHeader)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadSHA256)
	req.Header.Set("Content-Type", "application/gzip")
	req.ContentLength = fileSize

	// AWS SigV4 signing
	canonicalURI := "/" + s3Key
	if !strings.Contains(endpoint, "amazonaws.com") {
		canonicalURI = "/" + c.config.Bucket + "/" + s3Key
	}

	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		"application/gzip", hostHeader, payloadSHA256, amzDate)
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"

	canonicalReq := fmt.Sprintf("PUT\n%s\n\n%s\n%s\n%s",
		canonicalURI, canonicalHeaders, signedHeaders, payloadSHA256)

	reqHash := sha256.Sum256([]byte(canonicalReq))
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.config.Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, hex.EncodeToString(reqHash[:]))

	signingKey := getSignatureKey(c.config.SecretAccessKey, dateStamp, c.config.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.config.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed S3 upload network request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 upload returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// BackupUploadDestination helper
func (m *Manager) UploadToRemoteS3(backupMeta *BackupMetadata, config S3Config) error {
	client := NewS3Client(config)
	s3Key := filepath.Join("hostvra-backups", string(backupMeta.Type), backupMeta.FileName)
	return client.UploadFile(backupMeta.ArchivePath, s3Key)
}
