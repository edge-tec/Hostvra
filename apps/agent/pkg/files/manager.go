package files

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	ErrAccessDenied      = errors.New("access denied: path traversal or symlink escape detected outside allowed sandboxes")
	ErrFileAlreadyExists = errors.New("file or directory already exists")
	ErrInvalidFileName   = errors.New("invalid file or directory name")
	ErrRootDeletionBlocked  = errors.New("operation blocked: cannot delete sandbox root directory")
	ErrZipSlipDetected      = errors.New("security error: archive contains illegal relative path (Zip Slip attempt)")
	ErrArchiveBombDetected  = errors.New("archive decompression blocked: size or file count exceeds safety limits (archive bomb defense)")
	ErrSymlinkBlocked       = errors.New("security error: archive contains forbidden symlink or link entry")
)

const (
	MaxArchiveDecompressedBytes int64 = 1024 * 1024 * 1024 // 1GB limit
	MaxArchiveFileCount         int   = 20000              // 20,000 entries limit
)

type FileItem struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	PermOctal  string    `json:"perm_octal"`
	Owner      string    `json:"owner"`
	Group      string    `json:"group"`
	UID        int       `json:"uid"`
	GID        int       `json:"gid"`
	IsDir      bool      `json:"is_dir"`
	ModifiedAt time.Time `json:"modified_at"`
	Extension  string    `json:"extension"`
}

type FileManager struct {
	AllowedRoots []string
}

func NewFileManager(allowedRoots ...string) *FileManager {
	if len(allowedRoots) == 0 {
		// Production hosting sandboxes
		allowedRoots = []string{
			"/var/www",
			"/home",
			"/etc/nginx",
			"/etc/apache2",
			"/etc/php",
			"/var/log",
			"/tmp",
			"/root/Hostvra",
		}
	}

	cleaned := make([]string, 0, len(allowedRoots)*2)
	for _, r := range allowedRoots {
		cl := filepath.Clean(r)
		cleaned = append(cleaned, cl)
		if resolved, err := filepath.EvalSymlinks(cl); err == nil && resolved != cl {
			cleaned = append(cleaned, resolved)
		}
	}

	return &FileManager{
		AllowedRoots: cleaned,
	}
}

// AllowRootSystemWide explicitly enables root system directory access for root administrative operations
func (fm *FileManager) AllowRootSystemWide() {
	fm.AllowedRoots = []string{"/"}
}

// ValidatePath canonicalizes and ensures the path strictly resides within allowed sandboxes
func (fm *FileManager) ValidatePath(targetPath string) (string, error) {
	if strings.Contains(targetPath, "\x00") {
		return "", fmt.Errorf("%w: null byte injection detected", ErrAccessDenied)
	}

	clean := filepath.Clean(targetPath)

	// Step 1: Check against allowed roots
	withinRoot := false
	for _, root := range fm.AllowedRoots {
		if root == "/" || clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			withinRoot = true
			break
		}
	}
	if !withinRoot {
		return "", fmt.Errorf("%w: path %s is outside authorized roots", ErrAccessDenied, clean)
	}

	// Step 2: Symlink resolution boundary check (Prevent Symlink Escape Attacks)
	if _, err := os.Lstat(clean); err == nil {
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			return "", fmt.Errorf("%w: failed to evaluate symlinks: %v", ErrAccessDenied, err)
		}
		realWithinRoot := false
		for _, root := range fm.AllowedRoots {
			if root == "/" || resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator)) {
				realWithinRoot = true
				break
			}
		}
		if !realWithinRoot {
			return "", fmt.Errorf("%w: symlink targets outside authorized sandbox (%s -> %s)", ErrAccessDenied, clean, resolved)
		}
		return clean, nil
	}

	// Check parent directory if file does not exist yet (creation mode)
	parent := filepath.Dir(clean)
	if resolvedParent, err := filepath.EvalSymlinks(parent); err == nil {
		realWithinRoot := false
		for _, root := range fm.AllowedRoots {
			if root == "/" || resolvedParent == root || strings.HasPrefix(resolvedParent, root+string(filepath.Separator)) {
				realWithinRoot = true
				break
			}
		}
		if !realWithinRoot {
			return "", fmt.Errorf("%w: parent directory symlink targets outside authorized sandbox", ErrAccessDenied)
		}
	}

	return clean, nil
}

// List returns a sorted list of files and directories inside dirPath
func (fm *FileManager) List(dirPath string) ([]FileItem, error) {
	validated, err := fm.ValidatePath(dirPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(validated)
	if err != nil {
		return nil, err
	}

	items := make([]FileItem, 0, len(entries))
	for _, e := range entries {
		fullPath := filepath.Join(validated, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}

		item := buildFileItem(e.Name(), fullPath, info)
		items = append(items, item)
	}

	return items, nil
}

// Stat retrieves detailed file metadata for a specific path
func (fm *FileManager) Stat(filePath string) (*FileItem, error) {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(validated)
	if err != nil {
		return nil, err
	}

	item := buildFileItem(info.Name(), validated, info)
	return &item, nil
}

// ReadFile reads file contents up to maxBytes (safely capped to 10MB default for browser editors)
func (fm *FileManager) ReadFile(filePath string, maxBytes int64) ([]byte, error) {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(validated)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if maxBytes <= 0 || maxBytes > 25*1024*1024 {
		maxBytes = 10 * 1024 * 1024 // 10MB limit
	}

	return io.ReadAll(io.LimitReader(f, maxBytes))
}

// WriteFile atomically writes content to filePath, creating a .bak snapshot if overwriting
func (fm *FileManager) WriteFile(filePath string, content []byte) error {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(validated), 0755); err != nil {
		return err
	}

	// If file exists, create safety backup copy: <filename>.bak
	if _, err := os.Stat(validated); err == nil {
		bakPath := validated + ".bak"
		if existingBytes, readErr := os.ReadFile(validated); readErr == nil {
			_ = os.WriteFile(bakPath, existingBytes, 0644)
		}
	}

	// Atomic write: write to temporary file in same directory and rename
	tmpPath := fmt.Sprintf("%s.tmp.%d", validated, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, validated); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

// CreateDirectory creates a directory path recursively
func (fm *FileManager) CreateDirectory(dirPath string) error {
	validated, err := fm.ValidatePath(dirPath)
	if err != nil {
		return err
	}

	return os.MkdirAll(validated, 0755)
}

// Delete removes a file or directory tree safely
func (fm *FileManager) Delete(targetPath string) error {
	validated, err := fm.ValidatePath(targetPath)
	if err != nil {
		return err
	}

	for _, root := range fm.AllowedRoots {
		if validated == root {
			return ErrRootDeletionBlocked
		}
	}

	// 1. First attempt standard removal
	removeErr := os.RemoveAll(validated)
	if removeErr == nil {
		return nil
	}

	// 2. If RemoveAll failed (e.g. read-only subdirectories or restricted permissions),
	// attempt to make directories/files writable and retry
	_ = filepath.Walk(validated, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr == nil {
			if info.IsDir() {
				_ = os.Chmod(p, 0755)
			} else {
				_ = os.Chmod(p, 0644)
			}
		}
		return nil
	})

	if err := os.RemoveAll(validated); err == nil {
		return nil
	}

	// 3. Fallback to system command 'rm -rf' on Linux/Unix
	if execCmd := exec.Command("rm", "-rf", "--", validated); execCmd.Run() == nil {
		if _, statErr := os.Lstat(validated); os.IsNotExist(statErr) {
			return nil
		}
	}

	return removeErr
}

// Rename renames or moves a file/directory
func (fm *FileManager) Rename(oldPath, newPath string) error {
	valOld, err := fm.ValidatePath(oldPath)
	if err != nil {
		return err
	}

	valNew, err := fm.ValidatePath(newPath)
	if err != nil {
		return err
	}

	return os.Rename(valOld, valNew)
}

// Copy copies a file or directory tree to dstPath
func (fm *FileManager) Copy(srcPath, dstPath string) error {
	valSrc, err := fm.ValidatePath(srcPath)
	if err != nil {
		return err
	}

	valDst, err := fm.ValidatePath(dstPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(valSrc)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(valSrc, valDst)
	}

	return copyFile(valSrc, valDst)
}

// Chmod changes file permissions
func (fm *FileManager) Chmod(filePath string, mode os.FileMode) error {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return err
	}

	return os.Chmod(validated, mode)
}

// Chown changes owner and group
func (fm *FileManager) Chown(filePath string, uid, gid int) error {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return err
	}

	return os.Chown(validated, uid, gid)
}

// Archive compresses specified items into a .zip or .tar.gz archive
func (fm *FileManager) Archive(srcPaths []string, destArchivePath string, format string) error {
	valDest, err := fm.ValidatePath(destArchivePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(valDest), 0755); err != nil {
		return err
	}

	format = strings.ToLower(format)
	if format == "tar.gz" || strings.HasSuffix(valDest, ".tar.gz") || strings.HasSuffix(valDest, ".tgz") {
		return fm.archiveTarGz(srcPaths, valDest)
	}

	return fm.archiveZip(srcPaths, valDest)
}

func (fm *FileManager) archiveZip(srcPaths []string, destZip string) error {
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, src := range srcPaths {
		valSrc, err := fm.ValidatePath(src)
		if err != nil {
			return err
		}

		baseDir := filepath.Dir(valSrc)
		err = filepath.Walk(valSrc, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			if info.IsDir() {
				relPath += "/"
			}

			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relPath)
			header.Method = zip.Deflate

			w, err := zw.CreateHeader(header)
			if err != nil {
				return err
			}

			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(w, f); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (fm *FileManager) archiveTarGz(srcPaths []string, destTarGz string) error {
	out, err := os.Create(destTarGz)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, src := range srcPaths {
		valSrc, err := fm.ValidatePath(src)
		if err != nil {
			return err
		}

		baseDir := filepath.Dir(valSrc)
		err = filepath.Walk(valSrc, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relPath)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Extract extracts a .zip or .tar.gz archive with strict Zip Slip defense
func (fm *FileManager) Extract(archivePath string, destDir string) error {
	valArchive, err := fm.ValidatePath(archivePath)
	if err != nil {
		return err
	}

	valDest, err := fm.ValidatePath(destDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(valDest, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(valArchive, ".tar.gz") || strings.HasSuffix(valArchive, ".tgz") {
		return extractTarGz(valArchive, valDest)
	}

	return extractZip(valArchive, valDest)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) > MaxArchiveFileCount {
		return fmt.Errorf("%w: archive contains %d files (limit %d)", ErrArchiveBombDetected, len(r.File), MaxArchiveFileCount)
	}

	var totalExtractedBytes int64

	for _, f := range r.File {
		// Prevent symlink extraction attacks
		if f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: entry %s is a symlink", ErrSymlinkBlocked, f.Name)
		}

		cleanEntry := filepath.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		if strings.Contains(cleanEntry, "\x00") || strings.HasPrefix(cleanEntry, "/") {
			return fmt.Errorf("%w: invalid archive entry path %s", ErrZipSlipDetected, f.Name)
		}

		targetPath := filepath.Join(destDir, cleanEntry)

		// Zip Slip vulnerability check
		cleanTarget := filepath.Clean(targetPath)
		cleanDest := filepath.Clean(destDir)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return fmt.Errorf("%w: entry %s breaks out of destination", ErrZipSlipDetected, f.Name)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(cleanTarget, f.Mode())
			continue
		}

		// Prevent overwriting through pre-existing symlinks
		if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: destination entry %s is an existing symlink", ErrSymlinkBlocked, cleanTarget)
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return err
		}

		// Atomic extraction: stream to temporary file in same directory, then rename
		tmpTarget := fmt.Sprintf("%s.tmp.%d", cleanTarget, time.Now().UnixNano())
		outFile, err := os.OpenFile(tmpTarget, os.O_WRONLY|os.O_CREATE|os.O_EXCL, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			_ = os.Remove(tmpTarget)
			return err
		}

		// Cap per-file read to avoid zip bomb
		remainingAllowance := MaxArchiveDecompressedBytes - totalExtractedBytes
		if remainingAllowance <= 0 {
			rc.Close()
			outFile.Close()
			_ = os.Remove(tmpTarget)
			return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
		}

		written, err := io.Copy(outFile, io.LimitReader(rc, remainingAllowance+1))
		rc.Close()
		outFile.Close()
		if err != nil {
			_ = os.Remove(tmpTarget)
			return err
		}

		totalExtractedBytes += written
		if totalExtractedBytes > MaxArchiveDecompressedBytes {
			_ = os.Remove(tmpTarget)
			return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
		}

		// Re-verify destination before atomic rename to prevent TOCTOU race
		if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(tmpTarget)
			return fmt.Errorf("%w: destination entry %s is an existing symlink", ErrSymlinkBlocked, cleanTarget)
		}

		if err := os.Rename(tmpTarget, cleanTarget); err != nil {
			_ = os.Remove(tmpTarget)
			return err
		}
	}
	return nil
}

func extractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var totalExtractedBytes int64
	var fileCount int

	cleanDest := filepath.Clean(destDir)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileCount++
		if fileCount > MaxArchiveFileCount {
			return fmt.Errorf("%w: archive contains too many entries (limit %d)", ErrArchiveBombDetected, MaxArchiveFileCount)
		}

		// Strictly reject symlinks and hardlinks
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return fmt.Errorf("%w: tar entry %s is a link/symlink", ErrSymlinkBlocked, header.Name)
		}

		cleanEntry := filepath.Clean(strings.ReplaceAll(header.Name, "\\", "/"))
		if strings.Contains(cleanEntry, "\x00") || strings.HasPrefix(cleanEntry, "/") {
			return fmt.Errorf("%w: invalid archive entry path %s", ErrZipSlipDetected, header.Name)
		}

		targetPath := filepath.Join(destDir, cleanEntry)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) && cleanTarget != cleanDest {
			return fmt.Errorf("%w: entry %s breaks out of destination", ErrZipSlipDetected, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(cleanTarget, 0755)
		case tar.TypeReg:
			// Prevent overwriting through pre-existing symlinks
			if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: destination entry %s is an existing symlink", ErrSymlinkBlocked, cleanTarget)
			}

			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
				return err
			}

			// Atomic extraction: stream to temporary file in same directory, then rename
			tmpTarget := fmt.Sprintf("%s.tmp.%d", cleanTarget, time.Now().UnixNano())
			outFile, err := os.OpenFile(tmpTarget, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			remainingAllowance := MaxArchiveDecompressedBytes - totalExtractedBytes
			if remainingAllowance <= 0 {
				outFile.Close()
				_ = os.Remove(tmpTarget)
				return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
			}

			written, err := io.Copy(outFile, io.LimitReader(tr, remainingAllowance+1))
			outFile.Close()
			if err != nil {
				_ = os.Remove(tmpTarget)
				return err
			}

			totalExtractedBytes += written
			if totalExtractedBytes > MaxArchiveDecompressedBytes {
				_ = os.Remove(tmpTarget)
				return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
			}

			// Re-verify destination before atomic rename to prevent TOCTOU race
			if fi, err := os.Lstat(cleanTarget); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				_ = os.Remove(tmpTarget)
				return fmt.Errorf("%w: destination entry %s is an existing symlink", ErrSymlinkBlocked, cleanTarget)
			}

			if err := os.Rename(tmpTarget, cleanTarget); err != nil {
				_ = os.Remove(tmpTarget)
				return err
			}
		}
	}
	return nil
}

// buildFileItem parses file metadata, POSIX permissions, owner and group
func buildFileItem(name, path string, info os.FileInfo) FileItem {
	mode := info.Mode()
	octal := fmt.Sprintf("%04o", mode.Perm())
	ownerStr := "root"
	groupStr := "root"
	uid := 0
	gid := 0

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid = int(stat.Uid)
		gid = int(stat.Gid)
		if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
			ownerStr = u.Username
		} else {
			ownerStr = strconv.Itoa(uid)
		}
		if g, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil {
			groupStr = g.Name
		} else {
			groupStr = strconv.Itoa(gid)
		}
	}

	ext := strings.ToLower(filepath.Ext(name))

	return FileItem{
		Name:       name,
		Path:       path,
		Size:       info.Size(),
		Mode:       mode.String(),
		PermOctal:  octal,
		Owner:      ownerStr,
		Group:      groupStr,
		UID:        uid,
		GID:        gid,
		IsDir:      info.IsDir(),
		ModifiedAt: info.ModTime().UTC(),
		Extension:  ext,
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcSub := filepath.Join(src, entry.Name())
		dstSub := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcSub, dstSub); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcSub, dstSub); err != nil {
				return err
			}
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// ENTERPRISE FILE MANAGER v3.0 EXTENSIONS
// ----------------------------------------------------------------------------

type TreeNode struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	IsDir      bool        `json:"is_dir"`
	ChildCount int         `json:"child_count"`
	Children   []*TreeNode `json:"children,omitempty"`
}

// ListTree returns the immediate subdirectories for building a lazy tree
func (fm *FileManager) ListTree(targetPath string, showHidden bool) ([]*TreeNode, error) {
	validated, err := fm.ValidatePath(targetPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(validated)
	if err != nil {
		return nil, err
	}

	nodes := make([]*TreeNode, 0)
	for _, entry := range entries {
		name := entry.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			continue
		}

		fullSubPath := filepath.Join(validated, name)

		// Count immediate subdirectories inside this folder
		subCount := 0
		if subEntries, err := os.ReadDir(fullSubPath); err == nil {
			for _, se := range subEntries {
				if se.IsDir() && (showHidden || !strings.HasPrefix(se.Name(), ".")) {
					subCount++
				}
			}
		}

		nodes = append(nodes, &TreeNode{
			Name:       name,
			Path:       fullSubPath,
			IsDir:      true,
			ChildCount: subCount,
			Children:   nil,
		})
	}
	return nodes, nil
}

// CalculateDirSize recursively calculates size, total files and directories
func (fm *FileManager) CalculateDirSize(targetPath string) (int64, int, int, error) {
	validated, err := fm.ValidatePath(targetPath)
	if err != nil {
		return 0, 0, 0, err
	}

	var totalBytes int64
	var fileCount int
	var dirCount int

	err = filepath.Walk(validated, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip unreadable files gracefully
		}
		if info.IsDir() {
			if path != validated {
				dirCount++
			}
		} else {
			fileCount++
			totalBytes += info.Size()
		}
		return nil
	})

	return totalBytes, fileCount, dirCount, err
}

// ResolveConflictPath handles replace, skip, or auto-rename when target file already exists
func (fm *FileManager) ResolveConflictPath(destPath string, strategy string) (string, bool, error) {
	_, err := os.Lstat(destPath)
	if os.IsNotExist(err) {
		return destPath, false, nil
	}

	switch strings.ToLower(strategy) {
	case "replace":
		return destPath, false, nil
	case "skip":
		return destPath, true, nil
	case "rename":
		fallthrough
	default:
		dir := filepath.Dir(destPath)
		ext := filepath.Ext(destPath)
		base := strings.TrimSuffix(filepath.Base(destPath), ext)

		for i := 1; i <= 1000; i++ {
			candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
			if _, err := os.Lstat(candidate); os.IsNotExist(err) {
				return candidate, false, nil
			}
		}
		return "", false, fmt.Errorf("unable to generate unique conflict-free name for %s", destPath)
	}
}

// Search performs a recursive search matching query, file type filters, and size bounds
func (fm *FileManager) Search(rootPath string, query string, filterType string, minSize int64, maxSize int64, maxResults int) ([]FileItem, error) {
	validated, err := fm.ValidatePath(rootPath)
	if err != nil {
		return nil, err
	}

	if maxResults <= 0 {
		maxResults = 250
	}

	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	filterType = strings.ToLower(strings.TrimSpace(filterType))

	var results []FileItem

	err = filepath.Walk(validated, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip unreadable
		}
		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		name := info.Name()
		if lowerQuery != "" && !strings.Contains(strings.ToLower(name), lowerQuery) {
			return nil
		}

		size := info.Size()
		if minSize > 0 && size < minSize {
			return nil
		}
		if maxSize > 0 && size > maxSize {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		cleanExt := strings.TrimPrefix(ext, ".")

		if filterType != "" && filterType != "all" {
			match := false
			switch filterType {
			case "image":
				match = cleanExt == "jpg" || cleanExt == "jpeg" || cleanExt == "png" || cleanExt == "gif" || cleanExt == "webp" || cleanExt == "svg" || cleanExt == "ico"
			case "video":
				match = cleanExt == "mp4" || cleanExt == "webm" || cleanExt == "mov" || cleanExt == "avi" || cleanExt == "mkv"
			case "audio":
				match = cleanExt == "mp3" || cleanExt == "wav" || cleanExt == "ogg" || cleanExt == "m4a" || cleanExt == "flac"
			case "archive", "zip":
				match = cleanExt == "zip" || cleanExt == "tar" || cleanExt == "gz" || cleanExt == "bz2" || cleanExt == "rar" || cleanExt == "7z"
			case "php":
				match = cleanExt == "php" || cleanExt == "phtml" || cleanExt == "php8"
			case "html":
				match = cleanExt == "html" || cleanExt == "htm"
			case "css":
				match = cleanExt == "css" || cleanExt == "scss" || cleanExt == "less"
			case "js":
				match = cleanExt == "js" || cleanExt == "mjs" || cleanExt == "cjs" || cleanExt == "ts" || cleanExt == "jsx" || cleanExt == "tsx"
			case "json":
				match = cleanExt == "json"
			case "document":
				match = cleanExt == "pdf" || cleanExt == "doc" || cleanExt == "docx" || cleanExt == "txt" || cleanExt == "md" || cleanExt == "csv"
			}
			if !match {
				return nil
			}
		}

		var uid, gid int
		owner := "www-data"
		group := "www-data"
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			uid = int(stat.Uid)
			gid = int(stat.Gid)
			if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
				owner = u.Username
			}
			if g, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil {
				group = g.Name
			}
		}

		results = append(results, FileItem{
			Name:       name,
			Path:       path,
			Size:       size,
			Mode:       info.Mode().String(),
			PermOctal:  fmt.Sprintf("%04o", info.Mode().Perm()),
			Owner:      owner,
			Group:      group,
			UID:        uid,
			GID:        gid,
			IsDir:      info.IsDir(),
			ModifiedAt: info.ModTime(),
			Extension:  cleanExt,
		})

		return nil
	})

	return results, err
}

// MergeChunks assembles ordered chunk files into the final destination
func (fm *FileManager) MergeChunks(targetPath string, chunkPaths []string) error {
	validated, err := fm.ValidatePath(targetPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(validated), 0755); err != nil {
		return err
	}

	dest, err := os.Create(validated)
	if err != nil {
		return err
	}
	defer dest.Close()

	for _, chunkPath := range chunkPaths {
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(dest, chunkFile)
		chunkFile.Close()
		_ = os.Remove(chunkPath)
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// StreamZip archives the selected file paths and streams the zip output directly to w
func (fm *FileManager) StreamZip(paths []string, w io.Writer) error {
	archive := zip.NewWriter(w)
	defer archive.Close()

	for _, p := range paths {
		validated, err := fm.ValidatePath(p)
		if err != nil {
			continue
		}

		info, err := os.Stat(validated)
		if err != nil {
			continue
		}

		if info.IsDir() {
			baseDir := filepath.Dir(validated)
			_ = filepath.Walk(validated, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				relPath, err := filepath.Rel(baseDir, filePath)
				if err != nil {
					return nil
				}
				header, err := zip.FileInfoHeader(fileInfo)
				if err != nil {
					return nil
				}
				header.Name = filepath.ToSlash(relPath)
				if fileInfo.IsDir() {
					header.Name += "/"
				} else {
					header.Method = zip.Deflate
				}
				writer, err := archive.CreateHeader(header)
				if err != nil {
					return nil
				}
				if !fileInfo.IsDir() {
					f, err := os.Open(filePath)
					if err == nil {
						_, _ = io.Copy(writer, f)
						f.Close()
					}
				}
				return nil
			})
		} else {
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				continue
			}
			header.Name = filepath.Base(validated)
			header.Method = zip.Deflate
			writer, err := archive.CreateHeader(header)
			if err != nil {
				continue
			}
			f, err := os.Open(validated)
			if err == nil {
				_, _ = io.Copy(writer, f)
				f.Close()
			}
		}
	}
	return nil
}

