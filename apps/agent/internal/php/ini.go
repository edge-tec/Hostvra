package php

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// IniDirectiveDef represents a directive definition with category and defaults
type IniDirectiveDef struct {
	Directive   string `json:"directive"`
	Category    string `json:"category"`
	Type        string `json:"type"` // size, time, boolean, integer, string, enum
	Default     string `json:"default"`
	Recommended string `json:"recommended"`
	Description string `json:"description"`
}

// RecommendedStandardDirectives holds standard directives exposed in Simple Mode
var RecommendedStandardDirectives = []IniDirectiveDef{
	// Core
	{Directive: "memory_limit", Category: "Core", Type: "size", Default: "128M", Recommended: "512M", Description: "Maximum amount of memory a script may consume"},
	{Directive: "max_execution_time", Category: "Core", Type: "time", Default: "30", Recommended: "300", Description: "Maximum execution time of each script in seconds"},
	{Directive: "max_input_time", Category: "Core", Type: "time", Default: "60", Recommended: "300", Description: "Maximum amount of time each script may spend parsing request data"},
	{Directive: "max_input_vars", Category: "Core", Type: "integer", Default: "1000", Recommended: "5000", Description: "How many input variables may be accepted"},
	// Upload
	{Directive: "upload_max_filesize", Category: "Upload", Type: "size", Default: "2M", Recommended: "128M", Description: "Maximum allowed size for uploaded files"},
	{Directive: "post_max_size", Category: "Upload", Type: "size", Default: "8M", Recommended: "128M", Description: "Maximum size of POST data that PHP will accept"},
	{Directive: "max_file_uploads", Category: "Upload", Type: "integer", Default: "20", Recommended: "50", Description: "Maximum number of files that can be uploaded simultaneously"},
	// Error Handling
	{Directive: "display_errors", Category: "Error Handling", Type: "boolean", Default: "Off", Recommended: "Off", Description: "Print errors out as part of the output (Off for production)"},
	{Directive: "log_errors", Category: "Error Handling", Type: "boolean", Default: "On", Recommended: "On", Description: "Log errors to server log or error_log"},
	{Directive: "error_reporting", Category: "Error Handling", Type: "string", Default: "E_ALL & ~E_DEPRECATED & ~E_STRICT", Recommended: "E_ALL & ~E_DEPRECATED & ~E_STRICT", Description: "Level of error reporting"},
	// Date
	{Directive: "date.timezone", Category: "Date", Type: "string", Default: "UTC", Recommended: "UTC", Description: "Default timezone used by all date/time functions"},
	// OPcache
	{Directive: "opcache.enable", Category: "OPcache", Type: "boolean", Default: "1", Recommended: "1", Description: "Enables OPcache bytecode cache"},
	{Directive: "opcache.memory_consumption", Category: "OPcache", Type: "size", Default: "128", Recommended: "256", Description: "The OPcache shared memory storage size in MB"},
	{Directive: "opcache.max_accelerated_files", Category: "OPcache", Type: "integer", Default: "10000", Recommended: "20000", Description: "The maximum number of keys in the OPcache hash table"},
	{Directive: "opcache.validate_timestamps", Category: "OPcache", Type: "boolean", Default: "1", Recommended: "1", Description: "Validate file timestamps to invalidate cached files"},
	{Directive: "opcache.revalidate_freq", Category: "OPcache", Type: "time", Default: "2", Recommended: "2", Description: "How often in seconds to check file timestamps for changes"},
	// Security
	{Directive: "expose_php", Category: "Security", Type: "boolean", Default: "On", Recommended: "Off", Description: "Decides whether PHP may expose the fact that it is installed on the server"},
	{Directive: "allow_url_fopen", Category: "Security", Type: "boolean", Default: "On", Recommended: "On", Description: "Enables URL-aware fopen wrappers"},
	{Directive: "allow_url_include", Category: "Security", Type: "boolean", Default: "Off", Recommended: "Off", Description: "Enables include/require to open URLs (High security risk)"},
	{Directive: "disable_functions", Category: "Security", Type: "string", Default: "", Recommended: "exec,passthru,shell_exec,system,proc_open,popen", Description: "Disables dangerous system execution functions"},
	// Sessions
	{Directive: "session.gc_maxlifetime", Category: "Sessions", Type: "time", Default: "1440", Recommended: "1440", Description: "Number of seconds after which data will be seen as garbage and cleaned up"},
	{Directive: "session.cookie_httponly", Category: "Sessions", Type: "boolean", Default: "0", Recommended: "1", Description: "Whether to mark the session cookie as HTTP-only"},
	{Directive: "session.cookie_secure", Category: "Sessions", Type: "boolean", Default: "0", Recommended: "1", Description: "Whether cookies should only be sent over secure connections"},
}

// IniEditor parses, modifies, and validates php.ini files
type IniEditor struct{}

// NewIniEditor creates a new IniEditor
func NewIniEditor() *IniEditor {
	return &IniEditor{}
}

// ReadIniContent reads raw INI content from disk
func (e *IniEditor) ReadIniContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read ini file %s: %w", filePath, err)
	}
	return string(data), nil
}

// ParseDirectives extracts key/value directive pairs from an INI string
func (e *IniEditor) ParseDirectives(content string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	directiveRegex := regexp.MustCompile(`^\s*([a-zA-Z0-9_\.]+)\s*=\s*(.*?)\s*(?:;.*)?$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		matches := directiveRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := matches[1]
			val := matches[2]
			val = strings.Trim(val, `"'`)
			result[key] = val
		}
	}
	return result
}

// UpdateDirectives updates or appends directives into existing INI content preserving comments and structure
func (e *IniEditor) UpdateDirectives(originalContent string, updates map[string]string) string {
	lines := strings.Split(originalContent, "\n")
	updatedKeys := make(map[string]bool)

	var newLines []string
	directiveRegex := regexp.MustCompile(`^\s*(;?\s*)([a-zA-Z0-9_\.]+)\s*=\s*(.*?)\s*(;.*)?$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			newLines = append(newLines, line)
			continue
		}

		matches := directiveRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			key := matches[2]
			if newVal, exists := updates[key]; exists {
				comment := ""
				if len(matches) >= 5 && matches[4] != "" {
					comment = " " + matches[4]
				}
				newLines = append(newLines, fmt.Sprintf("%s = %s%s", key, newVal, comment))
				updatedKeys[key] = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// Append any directives not found in original INI
	var missing []string
	for k, v := range updates {
		if !updatedKeys[k] {
			missing = append(missing, fmt.Sprintf("%s = %s", k, v))
		}
	}

	if len(missing) > 0 {
		newLines = append(newLines, "", "; --- Hostvra Custom Directives ---")
		newLines = append(newLines, missing...)
	}

	return strings.Join(newLines, "\n")
}

// GenerateDiff returns a human-readable unified diff between before and after contents
func (e *IniEditor) GenerateDiff(before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	var diff bytes.Buffer
	diff.WriteString("--- Original Configuration\n+++ Proposed Configuration\n")

	beforeMap := e.ParseDirectives(before)
	afterMap := e.ParseDirectives(after)

	for k, oldVal := range beforeMap {
		if newVal, exists := afterMap[k]; exists {
			if oldVal != newVal {
				diff.WriteString(fmt.Sprintf("-%s = %s\n+%s = %s\n", k, oldVal, k, newVal))
			}
		} else {
			diff.WriteString(fmt.Sprintf("-%s = %s\n", k, oldVal))
		}
	}

	for k, newVal := range afterMap {
		if _, exists := beforeMap[k]; !exists {
			diff.WriteString(fmt.Sprintf("+%s = %s\n", k, newVal))
		}
	}

	_ = beforeLines
	_ = afterLines
	return diff.String()
}

// ValidatePath ensures the path is canonical and strictly within allowed PHP configuration directories
func (e *IniEditor) ValidatePath(targetPath string) error {
	cleanPath := filepath.Clean(targetPath)
	allowedPrefixes := []string{
		"/etc/php",
		"/etc/opt/remi",
		"/etc/php-fpm.d",
		"/etc/php.ini",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			return nil
		}
	}
	return fmt.Errorf("configuration path %s is outside allowed directories", targetPath)
}
