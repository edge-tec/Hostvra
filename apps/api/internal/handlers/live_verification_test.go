package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"hostvra/api/internal/auth"
)

func TestLivePostDeploymentVerification(t *testing.T) {
	baseURL := "http://127.0.0.1:8080"
	jwtSecret := "hostvra-production-secret-token-32-chars-long"

	// 1. Verify health endpoint
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("Live API server not responding at %s: %v", baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Health check returned status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var healthResp struct {
		Success bool `json:"success"`
		Data    struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &healthResp)
	if !healthResp.Success || healthResp.Data.Status != "healthy" {
		t.Fatalf("Health check response not healthy: %s", string(body))
	}
	t.Logf("LIVE HEALTH CHECK PASSED: Version %s, Status %s", healthResp.Data.Version, healthResp.Data.Status)

	// Setup Test Tenants
	orgA := uuid.New()
	userA := uuid.New()
	orgB := uuid.New()
	userB := uuid.New()

	pairA, _, err := auth.GenerateTokenPair(userA, orgA, "tenant-a@hostvra.live", "customer", false, jwtSecret, 24*time.Hour, 48*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token A: %v", err)
	}
	tokenA := pairA.AccessToken

	pairB, _, err := auth.GenerateTokenPair(userB, orgB, "tenant-b@hostvra.live", "customer", false, jwtSecret, 24*time.Hour, 48*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token B: %v", err)
	}
	tokenB := pairB.AccessToken

	client := &http.Client{Timeout: 5 * time.Second}

	// ------------------------------------------------------------------------
	// PHASE 11: LIVE TENANT ISOLATION TESTS
	// ------------------------------------------------------------------------
	t.Run("Live_TenantIsolation_DNS", func(t *testing.T) {
		// Tenant B queries non-existent or arbitrary zone ID
		fakeZoneID := uuid.New().String()
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/dns/zones/"+fakeZoneID+"/records", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer res.Body.Close()
		// Must be 404 or 403, never 200
		if res.StatusCode != http.StatusNotFound && res.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 404 or 403 for unauthorized zone, got %d", res.StatusCode)
		}
	})

	t.Run("Live_TenantIsolation_DatabasesAdmin", func(t *testing.T) {
		// Tenant B attempts to read root database password
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/databases/root-password", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer GetRootPassword, got %d", res.StatusCode)
		}

		// Tenant B attempts to get autobackup config
		reqAuto, _ := http.NewRequest("GET", baseURL+"/api/v1/databases/auto-backup", nil)
		reqAuto.Header.Set("Authorization", "Bearer "+tokenB)
		resAuto, err := client.Do(reqAuto)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resAuto.Body.Close()
		if resAuto.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer GetAutoBackup, got %d", resAuto.StatusCode)
		}
	})

	t.Run("Live_TenantIsolation_EmailAdmin", func(t *testing.T) {
		// Tenant B attempts to access email queue
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/email/queue", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer email queue, got %d", res.StatusCode)
		}
	})

	t.Run("Live_TenantIsolation_BackupSchedules", func(t *testing.T) {
		// Tenant B attempts full_config backup schedule
		schedPayload := map[string]interface{}{
			"name":       "Sneaky Backup",
			"scope":      "full_config",
			"target":     "hostvra",
			"frequency":  "daily",
			"keep_count": 3,
		}
		bodyBytes, _ := json.Marshal(schedPayload)
		req, _ := http.NewRequest("POST", baseURL+"/api/v1/backups/schedules", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+tokenB)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer full_config backup, got %d", res.StatusCode)
		}
	})

	// ------------------------------------------------------------------------
	// PHASE 13: FILESYSTEM LIVE ATTACK TESTS
	// ------------------------------------------------------------------------
	t.Run("Live_Filesystem_TraversalAndSystemProtection", func(t *testing.T) {
		attackPaths := []string{
			"/var/www/../etc/shadow",
			"/etc/shadow",
			"/root",
			"/proc/kcore",
			"/dev/mem",
			"/var/www/tenant-b/%2e%2e/%2e%2e/etc/passwd",
			"/var/www/tenant-b/%252e%252e/%252e%252e/etc/shadow",
		}

		for _, p := range attackPaths {
			req, _ := http.NewRequest("GET", baseURL+"/api/v1/files/content?path="+p, nil)
			req.Header.Set("Authorization", "Bearer "+tokenB)
			res, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				t.Fatalf("VULNERABILITY: Path %s returned 200 OK!", p)
			}
			if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusBadRequest {
				t.Fatalf("Expected 403 or 400 for path traversal %s, got %d", p, res.StatusCode)
			}
		}
	})

	// ------------------------------------------------------------------------
	// PHASE 14: TERMINAL LIVE ACCESS TESTS
	// ------------------------------------------------------------------------
	t.Run("Live_Terminal_CustomerBlocked", func(t *testing.T) {
		// 1. GetInfo
		reqInfo, _ := http.NewRequest("GET", baseURL+"/api/v1/terminal/info", nil)
		reqInfo.Header.Set("Authorization", "Bearer "+tokenB)
		resInfo, err := client.Do(reqInfo)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resInfo.Body.Close()
		if resInfo.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer terminal/info, got %d", resInfo.StatusCode)
		}

		// 2. Execute
		execPayload := map[string]string{
			"command": "whoami",
		}
		execBytes, _ := json.Marshal(execPayload)
		reqExec, _ := http.NewRequest("POST", baseURL+"/api/v1/terminal/execute", bytes.NewReader(execBytes))
		reqExec.Header.Set("Authorization", "Bearer "+tokenB)
		reqExec.Header.Set("Content-Type", "application/json")
		resExec, err := client.Do(reqExec)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resExec.Body.Close()
		if resExec.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for customer terminal/execute, got %d", resExec.StatusCode)
		}
	})

	// ------------------------------------------------------------------------
	// PHASE 15: INFORMATION LEAKAGE TESTS
	// ------------------------------------------------------------------------
	t.Run("Live_InformationLeakage_Inspection", func(t *testing.T) {
		// Unauthorized request inspection
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/databases/recycle-bin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenB)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer res.Body.Close()
		respBody, _ := io.ReadAll(res.Body)
		strBody := string(respBody)

		// Must not contain any sensitive internal system paths or root configs
		forbiddenStrings := []string{
			"/etc/shadow",
			"root_password",
			"private_key",
			"postgres://",
		}
		for _, fs := range forbiddenStrings {
			if bytes.Contains(respBody, []byte(fs)) {
				t.Fatalf("Information leak detected: response contains '%s': %s", fs, strBody)
			}
		}
	})

	// ------------------------------------------------------------------------
	// PHASE 12: CONCURRENT QUOTA RACE CONDITION TEST
	// ------------------------------------------------------------------------
	t.Run("Live_QuotaRace_ConcurrentRequests", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		results := make([]int, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				sitePayload := map[string]interface{}{
					"domain": fmt.Sprintf("race-site-%d-%s.com", idx, uuid.New().String()[:8]),
					"php":    "8.2",
				}
				body, _ := json.Marshal(sitePayload)
				req, _ := http.NewRequest("POST", baseURL+"/api/v1/websites", bytes.NewReader(body))
				req.Header.Set("Authorization", "Bearer "+tokenA)
				req.Header.Set("Content-Type", "application/json")
				res, err := client.Do(req)
				if err == nil {
					results[idx] = res.StatusCode
					res.Body.Close()
				}
			}(i)
		}
		wg.Wait()

		successCount := 0
		quotaCount := 0
		for _, code := range results {
			if code == http.StatusOK || code == http.StatusCreated {
				successCount++
			} else if code == http.StatusConflict {
				quotaCount++
			}
		}
		t.Logf("Live Quota Concurrency Results: Success=%d, QuotaExceeded=%d", successCount, quotaCount)
		// Crucial verification: Unlimited creations did NOT succeed
		if successCount > 1 {
			t.Fatalf("Quota race condition detected! %d sites created for starter plan", successCount)
		}
	})
}
