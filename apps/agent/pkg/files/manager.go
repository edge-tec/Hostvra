package files

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
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

	return os.RemoveAll(validated)
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

		outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// Cap per-file read to avoid zip bomb
		remainingAllowance := MaxArchiveDecompressedBytes - totalExtractedBytes
		if remainingAllowance <= 0 {
			rc.Close()
			outFile.Close()
			return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
		}

		written, err := io.Copy(outFile, io.LimitReader(rc, remainingAllowance+1))
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}

		totalExtractedBytes += written
		if totalExtractedBytes > MaxArchiveDecompressedBytes {
			return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
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
			outFile, err := os.OpenFile(cleanTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			remainingAllowance := MaxArchiveDecompressedBytes - totalExtractedBytes
			if remainingAllowance <= 0 {
				outFile.Close()
				return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
			}

			written, err := io.Copy(outFile, io.LimitReader(tr, remainingAllowance+1))
			outFile.Close()
			if err != nil {
				return err
			}

			totalExtractedBytes += written
			if totalExtractedBytes > MaxArchiveDecompressedBytes {
				return fmt.Errorf("%w: total extracted bytes exceeds safety limit (%d bytes)", ErrArchiveBombDetected, MaxArchiveDecompressedBytes)
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
