package wordpress

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceSerializedStrings(t *testing.T) {
	// Original serialized SQL snippet
	input := `INSERT INTO wp_options VALUES (1, 'siteurl', 's:19:"https://example.com";', 'yes');`
	expected := `INSERT INTO wp_options VALUES (1, 'siteurl', 's:27:"https://staging.example.com";', 'yes');`

	output := ReplaceSerializedStrings(input, "https://example.com", "https://staging.example.com")
	if output != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, output)
	}

	// Mixed serialized and non-serialized
	mixedInput := `s:19:"https://example.com"; and normal text https://example.com/blog`
	mixedOutput := ReplaceSerializedStrings(mixedInput, "https://example.com", "https://staging.example.com")
	if !strings.Contains(mixedOutput, `s:27:"https://staging.example.com";`) {
		t.Errorf("failed to replace serialized length: %s", mixedOutput)
	}
	if !strings.Contains(mixedOutput, `normal text https://staging.example.com/blog`) {
		t.Errorf("failed to replace plain text: %s", mixedOutput)
	}
}

func TestWordPress_CreateStagingAndPushToLive(t *testing.T) {
	tempDir := t.TempDir()
	liveRoot := filepath.Join(tempDir, "live")
	stagingRoot := filepath.Join(tempDir, "staging")

	_ = os.MkdirAll(filepath.Join(liveRoot, "wp-content"), 0755)
	_ = os.WriteFile(filepath.Join(liveRoot, "wp-config.php"), []byte("<?php\ndefine('DB_NAME', 'prod_db');\n"), 0644)
	_ = os.WriteFile(filepath.Join(liveRoot, "index.php"), []byte("<?php echo 'Production';"), 0644)
	_ = os.WriteFile(filepath.Join(liveRoot, "wp-content", "style.css"), []byte("/* v1.0 */"), 0644)

	mgr := NewManager()

	// 1. Create Staging
	createReq := CreateStagingRequest{
		LiveDocRoot:    liveRoot,
		LiveDomain:     "example.com",
		StagingDocRoot: stagingRoot,
		StagingDomain:  "staging.example.com",
		LiveDBName:     "prod_db",
		StagingDBName:  "staging_db",
		AuthUser:       "admin",
		AuthPassword:   "stg-secret-pass",
	}

	stgRes, err := mgr.CreateStaging(context.Background(), createReq)
	if err != nil {
		t.Fatalf("failed to create staging: %v", err)
	}
	if !stgRes.Success {
		t.Fatalf("staging creation not marked success")
	}

	// Verify Staging files and protections
	stgConfig, err := os.ReadFile(filepath.Join(stagingRoot, "wp-config.php"))
	if err != nil || !strings.Contains(string(stgConfig), "staging_db") {
		t.Errorf("expected staging_db in staging wp-config: %s", string(stgConfig))
	}
	if !strings.Contains(string(stgConfig), "WP_ENVIRONMENT_TYPE") {
		t.Errorf("expected WP_ENVIRONMENT_TYPE guard in staging wp-config")
	}

	robots, err := os.ReadFile(filepath.Join(stagingRoot, "robots.txt"))
	if err != nil || !strings.Contains(string(robots), "Disallow: /") {
		t.Errorf("expected robots.txt disallow in staging")
	}

	htpasswd, err := os.ReadFile(filepath.Join(stagingRoot, ".htpasswd"))
	if err != nil || !strings.Contains(string(htpasswd), "admin:stg-secret-pass") {
		t.Errorf("expected basic auth credentials in staging .htpasswd")
	}

	// 2. Modify staging file to simulate developer updates
	_ = os.WriteFile(filepath.Join(stagingRoot, "wp-content", "style.css"), []byte("/* v2.0 Staging Update */"), 0644)

	// 3. Push to Live
	pushReq := PushToLiveRequest{
		LiveDocRoot:    liveRoot,
		LiveDomain:     "example.com",
		StagingDocRoot: stagingRoot,
		StagingDomain:  "staging.example.com",
		LiveDBName:     "prod_db",
		StagingDBName:  "staging_db",
	}

	pushRes, err := mgr.PushToLive(context.Background(), pushReq)
	if err != nil {
		t.Fatalf("failed to push to live: %v", err)
	}
	if !pushRes.Success {
		t.Fatalf("push to live not marked success")
	}

	// Verify updated file in live
	liveStyle, _ := os.ReadFile(filepath.Join(liveRoot, "wp-content", "style.css"))
	if string(liveStyle) != "/* v2.0 Staging Update */" {
		t.Errorf("expected updated style in live, got: %s", string(liveStyle))
	}

	// Verify live wp-config.php maintained production DB name
	liveConfig, _ := os.ReadFile(filepath.Join(liveRoot, "wp-config.php"))
	if !strings.Contains(string(liveConfig), "prod_db") {
		t.Errorf("expected prod_db in live wp-config after push: %s", string(liveConfig))
	}

	// Verify safety backup was created
	if _, err := os.Stat(pushRes.BackupPath); os.IsNotExist(err) {
		t.Errorf("expected safety backup directory at %s", pushRes.BackupPath)
	}
}
