package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrPathTraversal = errors.New("path escapes workspace root")
	ErrPathAbsolute  = errors.New("absolute paths are not allowed for workspace-relative operations")
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
	for _, rel := range requiredLayoutDirs {
		abs, err := r.Resolve(rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("create layout directory %q: %w", rel, err)
		}
	}
	return nil
}

func (r Root) Resolve(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", errors.New("relative path is required")
	}
	if filepath.IsAbs(relPath) {
		return "", ErrPathAbsolute
	}

	clean := filepath.Clean(relPath)
	target := filepath.Join(r.Path, clean)
	relative, err := filepath.Rel(r.Path, target)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if strings.HasPrefix(relative, "..") || relative == ".." {
		return "", ErrPathTraversal
	}

	return target, nil
}

func (r Root) WriteFile(relPath string, content []byte) error {
	return r.WriteFileAtomic(relPath, content)
}

func (r Root) WriteFileAtomic(relPath string, content []byte) error {
	abs, err := r.Resolve(relPath)
	if err != nil {
		return err
	}
	if err := writeFileAtomic(abs, content, 0o644); err != nil {
		return fmt.Errorf("write file atomically: %w", err)
	}
	return nil
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

func writeFileAtomic(abs string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeWrite); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(abs)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
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

	if err := os.Rename(tmpPath, abs); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	removeTmp = false

	if err := maybeFailAtomicWrite(atomicWriteFaultBeforeDirSync); err != nil {
		return err
	}
	if err := syncDir(dir); err != nil {
		return err
	}
	if err := maybeFailAtomicWrite(atomicWriteFaultAfterDirSync); err != nil {
		return err
	}
	return nil
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
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
	abs, err := r.Resolve(relPath)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return content, nil
}
