package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const architectureSnapshotKind = "architecture-snapshot"

type architectureSnapshotManifest struct {
	Version     int                                `json:"version"`
	RunID       string                             `json:"run_id"`
	GeneratedAt string                             `json:"generated_at"`
	Files       []architectureSnapshotManifestFile `json:"files"`
}

type architectureSnapshotManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func persistPromotedArchitectureSnapshot(ws workspace.Root, runID string, now time.Time) (*Artifact, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("architecture snapshot run id is required")
	}
	files := []architectureSnapshotManifestFile{}
	for _, prefix := range []string{"model", "reports/as-is", "reports/diagrams", "reports/findings", "reports/coverage"} {
		source, err := ws.Resolve(prefix)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err = filepath.WalkDir(source, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("promoted architecture contains unsupported symlink %q", current)
			}
			if entry.IsDir() || !entry.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(ws.Path, current)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			content, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			target := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "promoted-snapshot", rel))
			if err := ws.WriteFile(target, content); err != nil {
				return err
			}
			digest := sha256.Sum256(content)
			files = append(files, architectureSnapshotManifestFile{Path: rel, SHA256: hex.EncodeToString(digest[:])})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := architectureSnapshotManifest{Version: 1, RunID: runID, GeneratedAt: now.UTC().Format(time.RFC3339), Files: files}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "promoted-snapshot", "architecture-snapshot.json"))
	if err := ws.WriteFile(manifestPath, append(raw, '\n')); err != nil {
		return nil, err
	}
	return &Artifact{Path: manifestPath, Kind: architectureSnapshotKind, Label: "Promoted architecture snapshot"}, nil
}
