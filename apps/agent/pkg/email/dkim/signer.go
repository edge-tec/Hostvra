package dkim

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type DKIMKeyPair struct {
	Domain        string
	Selector      string
	KeySize       int
	PrivateKeyPEM string
	PublicKeyDNS  string
}

// GenerateDKIMKey creates a 2048-bit RSA key pair and formats it for DNS TXT record
func GenerateDKIMKey(domain, selector string, keySize int) (*DKIMKeyPair, error) {
	if selector == "" {
		selector = "default"
	}
	if keySize == 0 {
		keySize = 2048
	}

	privKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privASN1 := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privASN1,
	}
	privPEM := string(pem.EncodeToMemory(privBlock))

	// Encode public key for DNS
	pubASN1, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubBase64 := base64.StdEncoding.EncodeToString(pubASN1)
	pubDNS := fmt.Sprintf("v=DKIM1; k=rsa; p=%s", pubBase64)

	return &DKIMKeyPair{
		Domain:        domain,
		Selector:      selector,
		KeySize:       keySize,
		PrivateKeyPEM: privPEM,
		PublicKeyDNS:  pubDNS,
	}, nil
}

// SaveDKIMKey writes the private and public keys to standard hostvra directory
func SaveDKIMKey(baseDir string, key *DKIMKeyPair) error {
	if baseDir == "" {
		baseDir = "/var/lib/hostvra/dkim"
	}
	domainDir := filepath.Join(baseDir, key.Domain)
	if err := os.MkdirAll(domainDir, 0700); err != nil {
		return fmt.Errorf("failed to create dkim directory: %w", err)
	}

	privPath := filepath.Join(domainDir, fmt.Sprintf("%s.private", key.Selector))
	if err := os.WriteFile(privPath, []byte(key.PrivateKeyPEM), 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	dnsPath := filepath.Join(domainDir, fmt.Sprintf("%s.txt", key.Selector))
	txtContent := fmt.Sprintf("%s._domainkey.%s IN TXT \"%s\"\n", key.Selector, key.Domain, key.PublicKeyDNS)
	if err := os.WriteFile(dnsPath, []byte(txtContent), 0644); err != nil {
		return fmt.Errorf("failed to write dns txt file: %w", err)
	}

	return nil
}

// SignOptions provides configuration parameters for RFC 6376 DKIM signing
type SignOptions struct {
	Domain        string
	Selector      string
	PrivateKeyPEM string
	Headers       []string // optional list of header names to sign
}

var whitespaceRegex = regexp.MustCompile(`[ \t]+`)

// CanonicalizeHeaderRelaxed applies RFC 6376 relaxed header canonicalization:
// 1. Lowercase header name
// 2. Unfold whitespace and collapse sequences of WSP to a single space
// 3. Delete whitespace before and after separating colon
func CanonicalizeHeaderRelaxed(name, value string) string {
	cleanName := strings.ToLower(strings.TrimSpace(name))
	cleanVal := whitespaceRegex.ReplaceAllString(strings.TrimSpace(value), " ")
	return fmt.Sprintf("%s:%s\r\n", cleanName, cleanVal)
}

// CanonicalizeBodyRelaxed applies RFC 6376 relaxed body canonicalization:
// 1. Reduce sequences of whitespace to a single space
// 2. Ignore all whitespace at end of lines
// 3. Remove all empty lines at the end of the body
// 4. End with a single CRLF (if non-empty)
func CanonicalizeBodyRelaxed(body string) string {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	var processed []string
	for _, l := range lines {
		// Reduce whitespace sequences and trim trailing whitespace
		collapsed := whitespaceRegex.ReplaceAllString(l, " ")
		trimmed := strings.TrimRight(collapsed, " \t")
		processed = append(processed, trimmed)
	}

	// Remove trailing empty lines
	for len(processed) > 0 && processed[len(processed)-1] == "" {
		processed = processed[:len(processed)-1]
	}

	if len(processed) == 0 {
		return ""
	}

	return strings.Join(processed, "\r\n") + "\r\n"
}

// parseRSAPrivateKey extracts rsa.PrivateKey from PEM data
func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	// Try PKCS#1
	if priv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return priv, nil
	}

	// Try PKCS#8
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if priv, ok := key.(*rsa.PrivateKey); ok {
			return priv, nil
		}
		return nil, errors.New("PKCS#8 key is not an RSA private key")
	}

	return nil, fmt.Errorf("unrecognized private key format: %w", err)
}

// SignEmail signs a raw RFC 5322 email message using RFC 6376 (rsa-sha256, relaxed/relaxed)
func SignEmail(rawMsg []byte, opts SignOptions) ([]byte, error) {
	if opts.Domain == "" {
		return nil, errors.New("DKIM signing requires domain")
	}
	if opts.Selector == "" {
		opts.Selector = "default"
	}
	if opts.PrivateKeyPEM == "" {
		return nil, errors.New("DKIM signing requires private key PEM")
	}

	privKey, err := parseRSAPrivateKey(opts.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DKIM private key: %w", err)
	}

	// Split headers and body
	var headerBytes, bodyBytes []byte
	if idx := bytes.Index(rawMsg, []byte("\r\n\r\n")); idx != -1 {
		headerBytes = rawMsg[:idx]
		bodyBytes = rawMsg[idx+4:]
	} else if idx := bytes.Index(rawMsg, []byte("\n\n")); idx != -1 {
		headerBytes = rawMsg[:idx]
		bodyBytes = rawMsg[idx+2:]
	} else {
		headerBytes = rawMsg
		bodyBytes = nil
	}

	// 1. Canonicalize Body
	canonBody := CanonicalizeBodyRelaxed(string(bodyBytes))
	bodyHash := sha256.Sum256([]byte(canonBody))
	bhBase64 := base64.StdEncoding.EncodeToString(bodyHash[:])

	// 2. Parse Raw Headers
	type HeaderField struct {
		Name  string
		Value string
	}
	var headers []HeaderField
	headerLines := strings.Split(strings.ReplaceAll(string(headerBytes), "\r\n", "\n"), "\n")
	for _, line := range headerLines {
		if line == "" {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(headers) > 0 {
			// Folded header
			headers[len(headers)-1].Value += " " + strings.TrimSpace(line)
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				headers = append(headers, HeaderField{
					Name:  strings.TrimSpace(parts[0]),
					Value: parts[1],
				})
			}
		}
	}

	// Default headers to sign if none specified
	targetHeaders := opts.Headers
	if len(targetHeaders) == 0 {
		targetHeaders = []string{"from", "to", "subject", "date", "message-id", "mime-version", "content-type"}
	}

	var signedHeaderNames []string
	var canonHeadersBuf bytes.Buffer

	for _, reqName := range targetHeaders {
		reqLower := strings.ToLower(reqName)
		// Find matching header (search backwards for multiple instances, RFC 6376 §5.4)
		for i := len(headers) - 1; i >= 0; i-- {
			if strings.ToLower(headers[i].Name) == reqLower {
				signedHeaderNames = append(signedHeaderNames, reqLower)
				canonHeadersBuf.WriteString(CanonicalizeHeaderRelaxed(headers[i].Name, headers[i].Value))
				break
			}
		}
	}

	if len(signedHeaderNames) == 0 {
		return nil, errors.New("no signable headers found in message")
	}

	// 3. Construct DKIM-Signature header skeleton
	hList := strings.Join(signedHeaderNames, ":")
	tVal := time.Now().Unix()
	dkimHeaderPrefix := fmt.Sprintf("v=1; a=rsa-sha256; c=relaxed/relaxed; d=%s; s=%s; t=%d; h=%s; bh=%s; b=",
		opts.Domain, opts.Selector, tVal, hList, bhBase64)

	// Canonicalize the DKIM-Signature header itself (without signature data)
	canonDKIMHeader := CanonicalizeHeaderRelaxed("dkim-signature", dkimHeaderPrefix)
	// Remove trailing CRLF from CanonicalizeHeaderRelaxed when signing the b= field RFC 6376 §3.5
	canonHeadersBuf.WriteString(strings.TrimRight(canonDKIMHeader, "\r\n"))

	// 4. Compute SHA-256 hash of canonicalized headers
	headersHash := sha256.Sum256(canonHeadersBuf.Bytes())

	// 5. Sign hash with RSA-SHA256
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, headersHash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to compute RSA signature: %w", err)
	}

	sigBase64 := base64.StdEncoding.EncodeToString(sigBytes)

	// Format complete DKIM-Signature header
	fullDKIMHeader := fmt.Sprintf("DKIM-Signature: %s%s\r\n", dkimHeaderPrefix, sigBase64)

	// Return fullDKIMHeader prepended to raw message
	var finalMsg bytes.Buffer
	finalMsg.WriteString(fullDKIMHeader)
	finalMsg.Write(rawMsg)

	return finalMsg.Bytes(), nil
}
