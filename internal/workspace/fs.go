package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	abs, err := r.Resolve(relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	if err := os.WriteFile(abs, content, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
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
