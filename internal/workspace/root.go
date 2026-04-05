package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ManifestFileName = "workspace.yaml"

var (
	ErrWorkspaceRequired = errors.New("workspace path is required")
	ErrWorkspaceAbsolute = errors.New("workspace path must be absolute")
	ErrWorkspaceNotDir   = errors.New("workspace path must point to a directory")
	ErrManifestMissing   = errors.New("workspace.yaml is required in workspace root")
)

type Root struct {
	Path         string
	ManifestPath string
	Manifest     Manifest
}

func Open(root string) (Root, error) {
	if root == "" {
		return Root{}, ErrWorkspaceRequired
	}
	if !filepath.IsAbs(root) {
		return Root{}, ErrWorkspaceAbsolute
	}

	info, err := os.Stat(root)
	if err != nil {
		return Root{}, fmt.Errorf("stat workspace: %w", err)
	}
	if !info.IsDir() {
		return Root{}, ErrWorkspaceNotDir
	}

	manifestPath := filepath.Join(root, ManifestFileName)
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Root{}, ErrManifestMissing
		}
		return Root{}, fmt.Errorf("stat manifest: %w", err)
	}
	if manifestInfo.IsDir() {
		return Root{}, ErrManifestMissing
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return Root{}, err
	}

	return Root{
		Path:         root,
		ManifestPath: manifestPath,
		Manifest:     manifest,
	}, nil
}
