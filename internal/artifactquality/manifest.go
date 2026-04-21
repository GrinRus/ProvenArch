package artifactquality

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const shardPackManifestFile = "shard-pack-manifest.json"

type ManifestAssessment struct {
	ManifestPresent           bool
	DocumentCount             int
	LinkedDocumentCount       int
	CitationCount             int
	RepoSpecificCitationCount int
	GenericRuntimeSummaryOnly bool
	Rich                      bool
}

func LoadManifestAssessment(writeRoot string) (ManifestAssessment, error) {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return ManifestAssessment{}, nil
	}
	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ManifestAssessment{}, nil
		}
		return ManifestAssessment{}, err
	}
	return AssessManifestBytes(raw)
}

func ValidateCollectManifestAtWriteRoot(writeRoot string) (ManifestAssessment, error) {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return ManifestAssessment{}, nil
	}
	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ManifestAssessment{}, nil
		}
		return ManifestAssessment{}, err
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return ManifestAssessment{}, err
	}
	assessment := AssessManifest(manifest)
	if err := validateManifestReadableSurface(writeRoot, manifest); err != nil {
		return assessment, err
	}
	return assessment, nil
}

func AssessManifestBytes(raw []byte) (ManifestAssessment, error) {
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return ManifestAssessment{}, err
	}
	return AssessManifest(manifest), nil
}

func AssessManifest(manifest contracts.ShardPackManifest) ManifestAssessment {
	assessment := ManifestAssessment{ManifestPresent: true}
	referencedCitationIDs := map[string]struct{}{}
	for _, document := range manifest.Documents {
		pathValue := strings.TrimSpace(document.Path)
		canonicalPathValue := strings.TrimSpace(document.CanonicalPath)
		if pathValue == "" && canonicalPathValue == "" {
			continue
		}
		assessment.DocumentCount++
		linkCount := 0
		for _, citationID := range document.CitationIDs {
			trimmedID := strings.TrimSpace(citationID)
			if trimmedID == "" {
				continue
			}
			referencedCitationIDs[trimmedID] = struct{}{}
			linkCount++
		}
		if linkCount > 0 {
			assessment.LinkedDocumentCount++
		}
	}

	uniqueLinkedCitationIDs := map[string]struct{}{}
	genericRuntimeSummaryOnly := true
	for _, citation := range manifest.Citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID == "" {
			continue
		}
		if _, linked := referencedCitationIDs[citationID]; !linked {
			continue
		}
		uniqueLinkedCitationIDs[citationID] = struct{}{}
		if !IsGenericRuntimeSummaryCitation(citationID) {
			genericRuntimeSummaryOnly = false
		}
		if strings.TrimSpace(citation.Repo) != "" &&
			strings.TrimSpace(citation.Path) != "" &&
			!IsGenericRuntimeSummaryCitation(citationID) {
			assessment.RepoSpecificCitationCount++
		}
	}

	assessment.CitationCount = len(uniqueLinkedCitationIDs)
	assessment.GenericRuntimeSummaryOnly = assessment.CitationCount > 0 && genericRuntimeSummaryOnly
	assessment.Rich = assessment.DocumentCount > 0 &&
		assessment.LinkedDocumentCount > 0 &&
		assessment.CitationCount > 0 &&
		assessment.RepoSpecificCitationCount > 0 &&
		!assessment.GenericRuntimeSummaryOnly
	return assessment
}

func HasRepoSpecificCitationSurface(manifest contracts.ShardPackManifest) bool {
	return AssessManifest(manifest).RepoSpecificCitationCount > 0
}

func validateManifestReadableSurface(writeRoot string, manifest contracts.ShardPackManifest) error {
	root := strings.TrimSpace(writeRoot)
	if root == "" {
		return nil
	}
	for _, document := range manifest.Documents {
		rel := strings.TrimSpace(document.Path)
		if rel == "" {
			return fmt.Errorf("document %q path is empty", strings.TrimSpace(document.ID))
		}
		candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
		relative, relErr := filepath.Rel(root, candidate)
		if relErr != nil {
			return fmt.Errorf("document %q path %q cannot be resolved: %w", strings.TrimSpace(document.ID), rel, relErr)
		}
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("document %q path %q escapes write_root", strings.TrimSpace(document.ID), rel)
		}
		info, statErr := os.Stat(candidate)
		if statErr != nil {
			return fmt.Errorf("document %q path %q is not readable under write_root: %w", strings.TrimSpace(document.ID), rel, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("document %q path %q points to a directory", strings.TrimSpace(document.ID), rel)
		}
	}
	return nil
}

func IsGenericRuntimeSummaryCitation(id string) bool {
	normalized := strings.ToLower(strings.TrimSpace(id))
	return normalized == "cite.runtime-summary" || strings.HasPrefix(normalized, "cite.runtime-summary.")
}

type WriteRootSnapshot struct {
	root      string
	backupDir string
	existed   bool
}

func SnapshotWriteRoot(root string) (*WriteRootSnapshot, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return &WriteRootSnapshot{}, nil
	}

	snapshot := &WriteRootSnapshot{root: root}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("write_root is not a directory: %s", root)
	}

	backupDir, err := os.MkdirTemp("", "acp-write-root-snapshot-*")
	if err != nil {
		return nil, err
	}
	snapshot.backupDir = backupDir
	snapshot.existed = true
	if err := copyDirectoryContents(root, backupDir); err != nil {
		_ = os.RemoveAll(backupDir)
		return nil, err
	}
	return snapshot, nil
}

func (s *WriteRootSnapshot) Restore() error {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return nil
	}
	if err := os.RemoveAll(s.root); err != nil {
		return err
	}
	if !s.existed {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	return copyDirectoryContents(s.backupDir, s.root)
}

func (s *WriteRootSnapshot) Cleanup() error {
	if s == nil || strings.TrimSpace(s.backupDir) == "" {
		return nil
	}
	return os.RemoveAll(s.backupDir)
}

func copyDirectoryContents(src string, dst string) error {
	entries := []string{}
	if err := filepath.Walk(strings.TrimSpace(src), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}
		entries = append(entries, path)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(entries)
	for _, path := range entries {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}
