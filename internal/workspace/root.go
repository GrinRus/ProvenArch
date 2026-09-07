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
	root         *os.Root
}

// Close releases a descriptor-backed subroot opened with OpenSubroot. Roots
// returned by Open do not own an open descriptor and therefore are no-ops.
func (r Root) Close() error {
	if r.root == nil {
		return nil
	}
	return r.root.Close()
}

// OpenSubroot opens a workspace-relative directory and keeps its descriptor
// alive for a sequence of related operations. Callers should use this when a
// check and a subsequent rename/remove/write must share one root identity.
func (r Root) OpenSubroot(relPath string) (Root, error) {
	clean, err := cleanRelativePath(relPath)
	if err != nil {
		return Root{}, err
	}
	parent, err := r.openFilesystemRoot()
	if err != nil {
		return Root{}, err
	}
	child, err := parent.OpenRoot(clean)
	if r.root == nil {
		_ = parent.Close()
	}
	if err != nil {
		return Root{}, err
	}
	return Root{Path: filepath.Join(r.Path, clean), root: child}, nil
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

	filesystemRoot := Root{Path: root}
	manifestPath := filepath.Join(root, ManifestFileName)
	osRoot, err := filesystemRoot.openFilesystemRoot()
	if err != nil {
		return Root{}, err
	}
	defer osRoot.Close()
	manifestInfo, err := osRoot.Stat(ManifestFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return Root{}, ErrManifestMissing
		}
		return Root{}, fmt.Errorf("stat manifest: %w", err)
	}
	if manifestInfo.IsDir() {
		return Root{}, ErrManifestMissing
	}

	rawManifest, err := osRoot.ReadFile(ManifestFileName)
	if err != nil {
		return Root{}, fmt.Errorf("read manifest: %w", err)
	}
	manifest, err := ParseManifest(rawManifest)
	if err != nil {
		return Root{}, err
	}

	return Root{
		Path:         root,
		ManifestPath: manifestPath,
		Manifest:     manifest,
	}, nil
}
