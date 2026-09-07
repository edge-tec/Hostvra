package ftp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidFTPUsername = errors.New("invalid FTP username: must be 3-32 characters, alphanumeric, underscores, and dots only")
	ErrInvalidDirectory   = errors.New("invalid FTP chroot home directory: path must be absolute and clean")
	ErrUserAlreadyExists  = errors.New("FTP user already exists")
	ErrUserNotFound       = errors.New("FTP user not found")
	ErrPureFtpdNotInstalled = errors.New("pure-ftpd or pure-pw tool is not installed")

	validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.]{3,32}$`)
)

type FTPUser struct {
	Username      string `json:"username"`
	HomeDir       string `json:"home_dir"`
	UID           int    `json:"uid"`
	GID           int    `json:"gid"`
	UploadBandwidth   int `json:"upload_bandwidth_kbps,omitempty"`   // in kB/s
	DownloadBandwidth int `json:"download_bandwidth_kbps,omitempty"` // in kB/s
	QuotaMB       int    `json:"quota_mb,omitempty"`
	MaxSessions   int    `json:"max_sessions,omitempty"`
	IsEnabled     bool   `json:"is_enabled"`
	CreatedAt     string `json:"created_at,omitempty"`
}

type FTPDaemonStatus struct {
	IsInstalled   bool   `json:"is_installed"`
	IsActive      bool   `json:"is_active"`
	DaemonName    string `json:"daemon_name"` // "pure-ftpd", "vsftpd", "dev"
	Port          int    `json:"port"`
	UsersCount    int    `json:"users_count"`
	ServerIP      string `json:"server_ip,omitempty"`
}

type FTPManager struct {
	mu           sync.RWMutex
	passwdFile   string
	pdbFile      string
	isPureFtpd   bool
}

func NewFTPManager() *FTPManager {
	passwdPath := "/etc/pure-ftpd/pureftpd.passwd"
	pdbPath := "/etc/pure-ftpd/pureftpd.pdb"

	isPureFtpd := false
	if _, err := exec.LookPath("pure-pw"); err == nil {
		isPureFtpd = true
	}

	// Fallback to local data dir if /etc/pure-ftpd is not writable or doesn't exist
	if _, err := os.Stat(filepath.Dir(passwdPath)); err != nil {
		home, _ := os.UserHomeDir()
		base := filepath.Join(home, ".hostvra", "ftp")
		_ = os.MkdirAll(base, 0755)
		passwdPath = filepath.Join(base, "pureftpd.passwd")
		pdbPath = filepath.Join(base, "pureftpd.pdb")
	}

	return &FTPManager{
		passwdFile: passwdPath,
		pdbFile:    pdbPath,
		isPureFtpd: isPureFtpd,
	}
}

// SetStoragePath allows customizing the virtual user passwd file (for testing)
func (fm *FTPManager) SetStoragePath(passwdPath string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.passwdFile = passwdPath
	fm.pdbFile = passwdPath + ".pdb"
	_ = os.MkdirAll(filepath.Dir(passwdPath), 0755)
}

// GetStatus checks daemon operational state
func (fm *FTPManager) GetStatus() (*FTPDaemonStatus, error) {
	active := false
	daemon := "pure-ftpd"

	if _, err := exec.LookPath("systemctl"); err == nil {
		out, err := exec.Command("systemctl", "is-active", "pure-ftpd").Output()
		if err == nil && strings.TrimSpace(string(out)) == "active" {
			active = true
		} else {
			// Check vsftpd
			outVs, errVs := exec.Command("systemctl", "is-active", "vsftpd").Output()
			if errVs == nil && strings.TrimSpace(string(outVs)) == "active" {
				active = true
				daemon = "vsftpd"
			}
		}
	} else {
		// Non-systemd or dev
		active = true
		daemon = "dev"
	}

	users, _ := fm.ListUsers()

	return &FTPDaemonStatus{
		IsInstalled: fm.isPureFtpd || daemon == "dev",
		IsActive:    active,
		DaemonName:  daemon,
		Port:        21,
		UsersCount:  len(users),
	}, nil
}

// ValidateUserParams validates username and directory
func (fm *FTPManager) ValidateUserParams(username, homeDir string) error {
	username = strings.TrimSpace(username)
	if !validUsernameRegex.MatchString(username) {
		return fmt.Errorf("%w: '%s'", ErrInvalidFTPUsername, username)
	}

	homeDir = filepath.Clean(strings.TrimSpace(homeDir))
	if !filepath.IsAbs(homeDir) || homeDir == "/" {
		return fmt.Errorf("%w: '%s'", ErrInvalidDirectory, homeDir)
	}

	return nil
}

// ListUsers reads all virtual FTP users
func (fm *FTPManager) ListUsers() ([]FTPUser, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	data, err := os.ReadFile(fm.passwdFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []FTPUser{}, nil
		}
		return nil, err
	}

	return fm.parsePasswd(string(data)), nil
}

// CreateUser registers a new virtual FTP user
func (fm *FTPManager) CreateUser(username, password, homeDir string, quotaMB, uploadBandwidth, downloadBandwidth int) (*FTPUser, error) {
	if err := fm.ValidateUserParams(username, homeDir); err != nil {
		return nil, err
	}
	if len(password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	// Ensure home directory exists
	_ = os.MkdirAll(homeDir, 0755)

	fm.mu.Lock()
	defer fm.mu.Unlock()

	existing, _ := fm.loadUsersUnlocked()
	for _, u := range existing {
		if strings.EqualFold(u.Username, username) {
			return nil, ErrUserAlreadyExists
		}
	}

	// In real Linux with pure-pw installed
	if fm.isPureFtpd {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "pure-pw", "useradd", username, "-u", "www-data", "-d", homeDir, "-m")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s\n", password, password))
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("pure-pw useradd failed: %s (%w)", string(out), err)
		}
		_ = exec.Command("pure-pw", "mkdb").Run()
	}

	newUser := FTPUser{
		Username:          username,
		HomeDir:           homeDir,
		UID:               33, // standard www-data UID
		GID:               33,
		QuotaMB:           quotaMB,
		UploadBandwidth:   uploadBandwidth,
		DownloadBandwidth: downloadBandwidth,
		IsEnabled:         true,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	existing = append(existing, newUser)
	if err := fm.saveUsersUnlocked(existing, username, password); err != nil {
		return nil, err
	}

	return &newUser, nil
}

// ChangePassword updates an FTP user's password
func (fm *FTPManager) ChangePassword(username, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("password must be at least 6 characters")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	existing, err := fm.loadUsersUnlocked()
	if err != nil {
		return err
	}

	found := false
	for _, u := range existing {
		if strings.EqualFold(u.Username, username) {
			found = true
			break
		}
	}

	if !found {
		return ErrUserNotFound
	}

	if fm.isPureFtpd {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "pure-pw", "passwd", username, "-m")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s\n%s\n", newPassword, newPassword))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("pure-pw passwd failed: %s (%w)", string(out), err)
		}
		_ = exec.Command("pure-pw", "mkdb").Run()
	}

	return fm.saveUsersUnlocked(existing, username, newPassword)
}

// UpdateUser modifies home dir, quota, bandwidth
func (fm *FTPManager) UpdateUser(username, homeDir string, quotaMB, uploadBandwidth, downloadBandwidth int) error {
	if err := fm.ValidateUserParams(username, homeDir); err != nil {
		return err
	}

	_ = os.MkdirAll(homeDir, 0755)

	fm.mu.Lock()
	defer fm.mu.Unlock()

	existing, err := fm.loadUsersUnlocked()
	if err != nil {
		return err
	}

	found := false
	for i, u := range existing {
		if strings.EqualFold(u.Username, username) {
			existing[i].HomeDir = homeDir
			existing[i].QuotaMB = quotaMB
			existing[i].UploadBandwidth = uploadBandwidth
			existing[i].DownloadBandwidth = downloadBandwidth
			found = true
			break
		}
	}

	if !found {
		return ErrUserNotFound
	}

	if fm.isPureFtpd {
		cmd := exec.Command("pure-pw", "usermod", username, "-d", homeDir, "-m")
		_ = cmd.Run()
		_ = exec.Command("pure-pw", "mkdb").Run()
	}

	return fm.saveUsersUnlocked(existing, "", "")
}

// DeleteUser removes an FTP user
func (fm *FTPManager) DeleteUser(username string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	existing, err := fm.loadUsersUnlocked()
	if err != nil {
		return err
	}

	newUsers := make([]FTPUser, 0, len(existing))
	found := false
	for _, u := range existing {
		if strings.EqualFold(u.Username, username) {
			found = true
			continue
		}
		newUsers = append(newUsers, u)
	}

	if !found {
		return ErrUserNotFound
	}

	if fm.isPureFtpd {
		cmd := exec.Command("pure-pw", "userdel", username, "-m")
		_ = cmd.Run()
		_ = exec.Command("pure-pw", "mkdb").Run()
	}

	return fm.saveUsersUnlocked(newUsers, "", "")
}

// ToggleUser enables or disables an FTP user
func (fm *FTPManager) ToggleUser(username string) (*FTPUser, error) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	existing, err := fm.loadUsersUnlocked()
	if err != nil {
		return nil, err
	}

	var target *FTPUser
	for i, u := range existing {
		if strings.EqualFold(u.Username, username) {
			existing[i].IsEnabled = !existing[i].IsEnabled
			target = &existing[i]
			break
		}
	}

	if target == nil {
		return nil, ErrUserNotFound
	}

	if err := fm.saveUsersUnlocked(existing, "", ""); err != nil {
		return nil, err
	}

	return target, nil
}

// ----------------------------------------------------------------------------
// PASSWD FILE PARSING AND ATOMIC PERSISTENCE
// ----------------------------------------------------------------------------

func (fm *FTPManager) loadUsersUnlocked() ([]FTPUser, error) {
	data, err := os.ReadFile(fm.passwdFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []FTPUser{}, nil
		}
		return nil, err
	}
	return fm.parsePasswd(string(data)), nil
}

func (fm *FTPManager) parsePasswd(content string) []FTPUser {
	lines := strings.Split(content, "\n")
	var users []FTPUser

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# DISABLED:")) {
			continue
		}

		isEnabled := true
		rawLine := line
		if strings.HasPrefix(line, "# DISABLED:") {
			isEnabled = false
			rawLine = strings.TrimSpace(strings.TrimPrefix(line, "# DISABLED:"))
		}

		parts := strings.Split(rawLine, ":")
		// pure-pw format: user:hash:uid:gid:gecos:homedir:upload:download:quota:ratio:time:etc
		if len(parts) >= 6 {
			uname := parts[0]
			uid, _ := strconv.Atoi(parts[2])
			gid, _ := strconv.Atoi(parts[3])
			homeDir := parts[5]

			u := FTPUser{
				Username:  uname,
				HomeDir:   homeDir,
				UID:       uid,
				GID:       gid,
				IsEnabled: isEnabled,
			}

			if len(parts) >= 8 {
				u.UploadBandwidth, _ = strconv.Atoi(parts[6])
				u.DownloadBandwidth, _ = strconv.Atoi(parts[7])
			}
			if len(parts) >= 9 {
				u.QuotaMB, _ = strconv.Atoi(parts[8])
			}

			users = append(users, u)
		}
	}

	return users
}

func (fm *FTPManager) saveUsersUnlocked(users []FTPUser, targetUser, targetPassword string) error {
	// Read existing hashes if available
	existingHashes := make(map[string]string)
	if data, err := os.ReadFile(fm.passwdFile); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "# DISABLED:")
			line = strings.TrimSpace(line)
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				existingHashes[parts[0]] = parts[1]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("# Hostvra Pure-FTPd Virtual User Database\n\n")

	for _, u := range users {
		prefix := ""
		if !u.IsEnabled {
			prefix = "# DISABLED: "
		}

		hash := existingHashes[u.Username]
		if u.Username == targetUser && targetPassword != "" {
			h := sha256.Sum256([]byte(targetPassword))
			hash = hex.EncodeToString(h[:])
		}
		if hash == "" {
			h := sha256.Sum256([]byte("default_secure_pass"))
			hash = hex.EncodeToString(h[:])
		}

		// pureftpd format: username:hash:uid:gid:gecos:homedir:upload:download:quota:...
		line := fmt.Sprintf("%s%s:%s:%d:%d::%s:%d:%d:%d::::::\n",
			prefix, u.Username, hash, u.UID, u.GID, u.HomeDir,
			u.UploadBandwidth, u.DownloadBandwidth, u.QuotaMB)
		sb.WriteString(line)
	}

	tmpFile := fm.passwdFile + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(sb.String()), 0600); err != nil {
		return err
	}

	return os.Rename(tmpFile, fm.passwdFile)
}
