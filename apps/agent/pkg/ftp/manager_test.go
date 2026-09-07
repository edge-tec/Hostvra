package ftp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFTP_Validation(t *testing.T) {
	fm := NewFTPManager()

	validUsers := []string{"webmaster", "site_user", "developer123", "deploy.bot"}
	for _, u := range validUsers {
		if err := fm.ValidateUserParams(u, "/var/www/site"); err != nil {
			t.Errorf("expected valid user %q to pass, got: %v", u, err)
		}
	}

	invalidUsers := []string{"ab", "user with spaces", "user;rm", "user/name", "a!b@c"}
	for _, u := range invalidUsers {
		if err := fm.ValidateUserParams(u, "/var/www/site"); err == nil {
			t.Errorf("expected invalid user %q to fail validation, but it passed", u)
		}
	}

	invalidDirs := []string{"", "relative/path", "/"}
	for _, d := range invalidDirs {
		if err := fm.ValidateUserParams("validuser", d); err == nil {
			t.Errorf("expected invalid directory %q to fail validation, but it passed", d)
		}
	}
}

func TestFTP_CRUDAndPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hostvra-ftp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	passwdFile := filepath.Join(tempDir, "pureftpd.passwd")
	fm := NewFTPManager()
	fm.SetStoragePath(passwdFile)

	siteHome := filepath.Join(tempDir, "site1")

	// 1. Create User
	u1, err := fm.CreateUser("user_site1", "Password123!", siteHome, 1024, 500, 1000)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if u1.Username != "user_site1" || u1.QuotaMB != 1024 || !u1.IsEnabled {
		t.Fatalf("unexpected user: %+v", u1)
	}

	// 2. Duplicate Check
	_, errDup := fm.CreateUser("user_site1", "AnotherPass123!", siteHome, 512, 0, 0)
	if errDup != ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got: %v", errDup)
	}

	// 3. List Users
	users, err := fm.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	// 4. Change Password
	err = fm.ChangePassword("user_site1", "NewSuperPassword123!")
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// 5. Update User
	err = fm.UpdateUser("user_site1", siteHome, 2048, 1000, 2000)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	usersUpdated, _ := fm.ListUsers()
	if usersUpdated[0].QuotaMB != 2048 {
		t.Fatalf("expected quota 2048, got %d", usersUpdated[0].QuotaMB)
	}

	// 6. Toggle User
	toggled, err := fm.ToggleUser("user_site1")
	if err != nil {
		t.Fatalf("ToggleUser failed: %v", err)
	}
	if toggled.IsEnabled {
		t.Fatalf("expected user to be disabled")
	}

	// 7. Delete User
	err = fm.DeleteUser("user_site1")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	finalUsers, _ := fm.ListUsers()
	if len(finalUsers) != 0 {
		t.Fatalf("expected 0 users after delete, got %d", len(finalUsers))
	}
}
