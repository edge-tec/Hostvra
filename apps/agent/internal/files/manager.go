package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrAccessDenied    = errors.New("access denied: path traversal attempt detected outside allowed sandbox")
	ErrFileAlreadyExists = errors.New("file or directory already exists")
	ErrInvalidFileName = errors.New("invalid file or directory name")
)

type FileItem struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	IsDir      bool      `json:"is_dir"`
	ModifiedAt time.Time `json:"modified_at"`
}

type FileManager struct {
	AllowedRoots []string
}

func NewFileManager(allowedRoots ...string) *FileManager {
	if len(allowedRoots) == 0 {
		// Default secure roots for web hosting and app deployments
		allowedRoots = []string{"/var/www", "/var/lib/hostvra", "/tmp/hostvra-test"}
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

func (fm *FileManager) ValidatePath(targetPath string) (string, error) {
	clean := filepath.Clean(targetPath)

	for _, root := range fm.AllowedRoots {
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			return clean, nil
		}
	}

	if resolved, err := filepath.EvalSymlinks(clean); err == nil {
		for _, root := range fm.AllowedRoots {
			if resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator)) {
				return clean, nil
			}
		}
	}

	return "", fmt.Errorf("%w: %s not within authorized sandboxes", ErrAccessDenied, clean)
}

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
		info, err := e.Info()
		if err != nil {
			continue
		}

		item := FileItem{
			Name:       e.Name(),
			Path:       filepath.Join(dirPath, e.Name()),
			Size:       info.Size(),
			Mode:       info.Mode().String(),
			IsDir:      e.IsDir(),
			ModifiedAt: info.ModTime().UTC(),
		}
		items = append(items, item)
	}

	return items, nil
}

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

	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024 // 5MB default limit for editor
	}

	return io.ReadAll(io.LimitReader(f, maxBytes))
}

func (fm *FileManager) WriteFile(filePath string, content []byte) error {
	validated, err := fm.ValidatePath(filePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(validated), 0755); err != nil {
		return err
	}

	return os.WriteFile(validated, content, 0644)
}

func (fm *FileManager) CreateDirectory(dirPath string) error {
	validated, err := fm.ValidatePath(dirPath)
	if err != nil {
		return err
	}

	return os.MkdirAll(validated, 0755)
}

func (fm *FileManager) Delete(targetPath string) error {
	validated, err := fm.ValidatePath(targetPath)
	if err != nil {
		return err
	}

	for _, root := range fm.AllowedRoots {
		if validated == root {
			return errors.New("cannot delete sandbox root directory")
		}
	}

	return os.RemoveAll(validated)
}

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
