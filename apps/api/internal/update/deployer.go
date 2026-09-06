package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrPathTraversalAttack = errors.New("archive contains path traversal element targeting outside release directory")
	ErrReleaseNotFound     = errors.New("target release directory does not exist")
)

// ReleaseDeployer manages versioned release staging, atomic symlink switching, and safe rollbacks
type ReleaseDeployer struct {
	baseDir        string
	releasesDir    string
	currentSymlink string
}

// NewReleaseDeployer creates a new ReleaseDeployer instance
func NewReleaseDeployer(baseDir string) *ReleaseDeployer {
	if baseDir == "" {
		baseDir = "/opt/hostvra"
	}
	return &ReleaseDeployer{
		baseDir:        baseDir,
		releasesDir:    filepath.Join(baseDir, "releases"),
		currentSymlink: filepath.Join(baseDir, "current"),
	}
}

// StageRelease extracts a verified tar.gz release package into /opt/hostvra/releases/<version>/
func (d *ReleaseDeployer) StageRelease(ctx context.Context, version string, archiveReader io.Reader) (string, error) {
	releasePath := filepath.Join(d.releasesDir, version)

	// Create release destination
	if err := os.MkdirAll(releasePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create release directory: %w", err)
	}

	gr, err := gzip.NewReader(archiveReader)
	if err != nil {
		return "", fmt.Errorf("invalid gzip archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	cleanDest := filepath.Clean(releasePath)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("corrupted tar entry: %w", err)
		}
		if hdr == nil {
			continue
		}

		// Security Check: Path Traversal Protection
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "../") || strings.Contains(cleanName, "/../") {
			return "", fmt.Errorf("%w: entry %q", ErrPathTraversalAttack, hdr.Name)
		}

		target := filepath.Join(cleanDest, cleanName)
		if !strings.HasPrefix(target, cleanDest+string(filepath.Separator)) && target != cleanDest {
			return "", fmt.Errorf("%w: target %q escapes %q", ErrPathTraversalAttack, target, cleanDest)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent dir: %w", err)
			}

			mode := os.FileMode(hdr.Mode)
			if mode&0111 != 0 {
				mode = 0755 // Executable binary
			} else {
				mode = 0644 // Regular file
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
			if err != nil {
				return "", fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return "", fmt.Errorf("failed to write file content %s: %w", target, err)
			}
			_ = f.Close()
		case tar.TypeSymlink:
			// Ensure symlink does not escape
			linkTarget := hdr.Linkname
			if strings.HasPrefix(linkTarget, "/") || strings.Contains(linkTarget, "../") {
				return "", fmt.Errorf("%w: unsafe symlink %s -> %s", ErrPathTraversalAttack, target, linkTarget)
			}
			_ = os.Remove(target)
			if err := os.Symlink(linkTarget, target); err != nil {
				return "", fmt.Errorf("failed to create symlink %s -> %s: %w", target, linkTarget, err)
			}
		}
	}

	return releasePath, nil
}

// ActivateRelease atomically switches the /opt/hostvra/current symlink to the target release
func (d *ReleaseDeployer) ActivateRelease(ctx context.Context, version string) error {
	targetReleasePath := filepath.Join(d.releasesDir, version)
	if _, err := os.Stat(targetReleasePath); err != nil {
		return fmt.Errorf("%w: %s", ErrReleaseNotFound, targetReleasePath)
	}

	// Create temporary symlink in the same parent directory to allow atomic rename
	tempLink := fmt.Sprintf("%s.tmp.%d", d.currentSymlink, os.Getpid())
	_ = os.Remove(tempLink)

	if err := os.Symlink(targetReleasePath, tempLink); err != nil {
		return fmt.Errorf("failed to create temporary symlink: %w", err)
	}

	// Atomic Rename
	if err := os.Rename(tempLink, d.currentSymlink); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("failed to atomically switch current symlink to %s: %w", version, err)
	}

	return nil
}

// RollbackRelease switches the /opt/hostvra/current symlink back to a previous verified version
func (d *ReleaseDeployer) RollbackRelease(ctx context.Context, previousVersion string) error {
	return d.ActivateRelease(ctx, previousVersion)
}

// GetCurrentActiveVersion inspects /opt/hostvra/current symlink to identify active version
func (d *ReleaseDeployer) GetCurrentActiveVersion(ctx context.Context) (string, error) {
	linkTarget, err := os.Readlink(d.currentSymlink)
	if err != nil {
		return "", err
	}
	return filepath.Base(linkTarget), nil
}

// CleanupOldReleases keeps the current active version, the previous rollback version, and up to keepCount releases
func (d *ReleaseDeployer) CleanupOldReleases(ctx context.Context, keepCount int) error {
	entries, err := os.ReadDir(d.releasesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	currentActive, _ := d.GetCurrentActiveVersion(ctx)

	var releases []string
	for _, e := range entries {
		if e.IsDir() {
			releases = append(releases, e.Name())
		}
	}

	if len(releases) <= keepCount {
		return nil
	}

	sort.Strings(releases)

	// Keep newest keepCount
	toDelete := releases[:len(releases)-keepCount]
	for _, rel := range toDelete {
		if rel == currentActive {
			continue // Never delete active
		}
		_ = os.RemoveAll(filepath.Join(d.releasesDir, rel))
	}

	return nil
}
