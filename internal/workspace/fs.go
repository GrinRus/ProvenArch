package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	ErrPathTraversal = errors.New("path escapes workspace root")
	ErrPathAbsolute  = errors.New("absolute paths are not allowed for workspace-relative operations")
	ErrPathSymlink   = errors.New("workspace path resolves through an unsafe symlink")
	ErrFileTooLarge  = errors.New("workspace file exceeds read limit")
)

var requiredLayoutDirs = []string{
	"charter/cards/domains",
	"charter/cards/teams",
	"charter/templates",
	"skills",
	"model/entities",
	"model/edges",
	"reports/as-is/services",
	"reports/findings",
	"reports/coverage",
	"reports/taskruns",
	"reports/agent-outputs/domains",
	"reports/agent-outputs/architect",
	"reports/changelog",
	"proposals",
	"docs/imports",
	"docs/rfcs",
	"docs/meetings",
	"docs/decisions",
}

func (r Root) EnsureLayout() error {
	root, err := r.openFilesystemRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	for _, rel := range requiredLayoutDirs {
		if err := root.MkdirAll(filepath.FromSlash(rel), 0o755); err != nil {
			return fmt.Errorf("create layout directory %q: %w", rel, err)
		}
	}
	return nil
}

func (r Root) Resolve(relPath string) (string, error) {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return "", err
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return "", err
	}
	defer root.Close()
	if err := validateRootPath(root, clean); err != nil {
		return "", err
	}
	return filepath.Join(r.Path, clean), nil
}

func cleanRelativePath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", errors.New("relative path is required")
	}
	if strings.ContainsRune(relPath, 0) {
		return "", errors.New("relative path contains NUL")
	}
	if filepath.IsAbs(relPath) {
		return "", ErrPathAbsolute
	}

	clean := filepath.Clean(relPath)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}
	return clean, nil
}

func (r Root) openFilesystemRoot() (*os.Root, error) {
	root, err := os.OpenRoot(r.Path)
	if err != nil {
		return nil, fmt.Errorf("open workspace root: %w", err)
	}
	return root, nil
}

// validateRootPath asks os.Root to resolve every existing component. A missing
// final component is valid only when its existing parent resolves inside root.
func validateRootPath(root *os.Root, clean string) error {
	current := clean
	for {
		_, err := root.Stat(current)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %v", ErrPathSymlink, err)
		}
		if info, linkErr := root.Lstat(current); linkErr == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				return nil
			}
			if _, retryErr := root.Stat(current); retryErr == nil {
				return nil
			} else {
				return fmt.Errorf("%w: dangling symlink %q: %v", ErrPathSymlink, current, retryErr)
			}
		} else if !errors.Is(linkErr, os.ErrNotExist) {
			return fmt.Errorf("%w: %v", ErrPathSymlink, linkErr)
		}
		parent := filepath.Dir(current)
		if parent == current || parent == "." {
			return nil
		}
		current = parent
	}
}

func (r Root) WriteFile(relPath string, content []byte) error {
	return r.WriteFileAtomic(relPath, content)
}

func (r Root) WriteFileAtomic(relPath string, content []byte) error {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return err
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	if err := writeFileAtomic(root, clean, content, 0o644); err != nil {
		return fmt.Errorf("write file atomically: %w", err)
	}
	return nil
}

// WriteDirectoryAtomicExclusive publishes a complete directory without exposing
// partially written files and never replaces an existing destination.
func (r Root) WriteDirectoryAtomicExclusive(relDir string, files map[string][]byte) error {
	cleanDir, err := cleanRelativePath(relDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("directory files are required")
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return err
	}
	defer root.Close()

	parent := filepath.Dir(cleanDir)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create directory parent: %w", err)
	}
	lockPath := cleanDir + ".lock"
	lock, err := root.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fs.ErrExist
		}
		return fmt.Errorf("reserve directory publication: %w", err)
	}
	_ = lock.Close()
	defer root.Remove(lockPath)
	if _, err := root.Stat(cleanDir); err == nil {
		return fs.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat directory destination: %w", err)
	}

	tmpDir := cleanDir + ".tmp-" + randomHex(8)
	if err := root.Mkdir(tmpDir, 0o755); err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = root.RemoveAll(tmpDir)
		}
	}()

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cleanName, err := cleanRelativePath(name)
		if err != nil || cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid directory file %q", name)
		}
		target := filepath.Join(tmpDir, cleanName)
		if err := root.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create temporary file parent: %w", err)
		}
		if err := maybeFailAtomicWrite(atomicWriteFaultBeforeWrite); err != nil {
			return err
		}
		handle, err := root.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("create temporary file %q: %w", name, err)
		}
		if _, err := handle.Write(files[name]); err != nil {
			_ = handle.Close()
			return fmt.Errorf("write temporary file %q: %w", name, err)
		}
		if err := handle.Sync(); err != nil {
			_ = handle.Close()
			return fmt.Errorf("sync temporary file %q: %w", name, err)
		}
		if err := handle.Close(); err != nil {
			return fmt.Errorf("close temporary file %q: %w", name, err)
		}
	}
	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeRename); err != nil {
		return err
	}
	if err := root.Rename(tmpDir, cleanDir); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fs.ErrExist
		}
		return fmt.Errorf("publish directory: %w", err)
	}
	removeTmp = false
	return nil
}

func randomHex(bytesCount int) string {
	raw := make([]byte, bytesCount)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(raw)
}

func (r Root) WriteFileAtomicWithLastGood(relPath string, content []byte) error {
	if err := r.WriteFileAtomic(relPath, content); err != nil {
		return err
	}
	if err := r.WriteFileAtomic(lastGoodRelPath(relPath), content); err != nil {
		return fmt.Errorf("write last-good copy: %w", err)
	}
	return nil
}

func (r Root) ReadFileWithLastGood(relPath string) (content []byte, recoveredFromLastGood bool, err error) {
	content, err = r.ReadFile(relPath)
	if err == nil {
		return content, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		lastGood, lastGoodErr := r.ReadLastGoodFile(relPath)
		if lastGoodErr == nil {
			return lastGood, true, nil
		}
		return nil, false, err
	}
	lastGood, lastGoodErr := r.ReadLastGoodFile(relPath)
	if lastGoodErr == nil {
		return lastGood, true, nil
	}
	if errors.Is(lastGoodErr, os.ErrNotExist) {
		return nil, false, err
	}
	return nil, false, lastGoodErr
}

func (r Root) ReadLastGoodFile(relPath string) ([]byte, error) {
	return r.ReadFile(lastGoodRelPath(relPath))
}

func lastGoodRelPath(relPath string) string {
	return relPath + ".last-good"
}

type atomicWriteFaultPoint string

const (
	atomicWriteFaultBeforeWrite   atomicWriteFaultPoint = "before_write"
	atomicWriteFaultAfterWrite    atomicWriteFaultPoint = "after_write"
	atomicWriteFaultBeforeRename  atomicWriteFaultPoint = "before_rename"
	atomicWriteFaultBeforeDirSync atomicWriteFaultPoint = "before_dir_sync"
	atomicWriteFaultAfterDirSync  atomicWriteFaultPoint = "after_dir_sync"
)

var atomicWriteFaults struct {
	sync.Mutex
	hook func(atomicWriteFaultPoint) error
}

func setAtomicWriteFaultHookForTest(hook func(atomicWriteFaultPoint) error) func() {
	atomicWriteFaults.Lock()
	previous := atomicWriteFaults.hook
	atomicWriteFaults.hook = hook
	atomicWriteFaults.Unlock()
	return func() {
		atomicWriteFaults.Lock()
		atomicWriteFaults.hook = previous
		atomicWriteFaults.Unlock()
	}
}

func maybeFailAtomicWrite(point atomicWriteFaultPoint) error {
	atomicWriteFaults.Lock()
	hook := atomicWriteFaults.hook
	atomicWriteFaults.Unlock()
	if hook == nil {
		return nil
	}
	return hook(point)
}

func writeFileAtomic(root *os.Root, relPath string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(relPath)
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeWrite); err != nil {
		return err
	}

	tmp, tmpPath, err := createRootTemp(root, dir, filepath.Base(relPath), perm)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = root.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := maybeFailAtomicWrite(atomicWriteFaultAfterWrite); err != nil {
		return err
	}
	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeRename); err != nil {
		return err
	}

	if err := root.Rename(tmpPath, relPath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	removeTmp = false

	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeDirSync); err != nil {
		return err
	}
	if err := syncRootDir(root, dir); err != nil {
		return err
	}
	if err := maybeFailAtomicWrite(atomicWriteFaultAfterDirSync); err != nil {
		return err
	}
	return nil
}

func createRootTemp(root *os.Root, dir string, base string, perm os.FileMode) (*os.File, string, error) {
	for attempt := 0; attempt < 100; attempt++ {
		var random [8]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, "", fmt.Errorf("generate temp suffix: %w", err)
		}
		name := "." + base + ".tmp-" + hex.EncodeToString(random[:])
		rel := filepath.Join(dir, name)
		handle, err := root.OpenFile(rel, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if err == nil {
			return handle, rel, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", errors.New("exhausted atomic temp file attempts")
}

func syncRootDir(root *os.Root, dir string) error {
	handle, err := root.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent directory for sync: %w", err)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync parent directory: %w", err)
	}
	return nil
}

func (r Root) ReadFile(relPath string) ([]byte, error) {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return nil, err
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	content, err := root.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return content, nil
}

func (r Root) ReadFileLimit(relPath string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errors.New("positive read limit is required")
	}
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return nil, err
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("open limited file: %w", err)
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read limited file: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("%w: %d bytes", ErrFileTooLarge, maxBytes)
	}
	return content, nil
}

type TreeFile struct {
	Path    string
	Content []byte
}

// ReadRegularTree returns a deterministic, immutable view of regular files
// below relRoot. Symlinks and special files are rejected instead of followed.
func (r Root) ReadRegularTree(relRoot string) ([]TreeFile, error) {
	clean, err := cleanRelativePath(relRoot)
	if err != nil {
		return nil, err
	}
	root, err := r.openFilesystemRoot()
	if err != nil {
		return nil, err
	}
	defer root.Close()

	files := []TreeFile{}
	err = fs.WalkDir(root.FS(), clean, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == clean {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: tree contains symlink %q", ErrPathSymlink, name)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("workspace tree contains non-regular file %q", name)
		}
		raw, err := root.ReadFile(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(clean, name)
		if err != nil {
			return err
		}
		files = append(files, TreeFile{Path: filepath.ToSlash(rel), Content: append([]byte(nil), raw...)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read workspace tree: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
