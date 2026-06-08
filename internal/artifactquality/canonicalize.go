package artifactquality

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func ValidateCollectManifestInRoot(writeRoot string) error {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return fmt.Errorf("collect write_root is empty")
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return err
	}
	return validateCollectManifestDocumentFiles(writeRoot, manifest.Documents)
}

func ValidateCollectManifestBytes(raw []byte) error {
	_, err := contracts.ParseShardPackManifest(raw)
	return err
}

func validateCollectManifestDocumentFiles(writeRoot string, documents []contracts.AuthoredDocument) error {
	root := filepath.Clean(writeRoot)
	problems := []string{}
	for idx, document := range documents {
		label := fmt.Sprintf("documents[%d].path", idx)
		rawPath := strings.TrimSpace(document.Path)
		if rawPath == "" {
			problems = append(problems, label+" references an empty document file path")
			continue
		}
		cleanRel := filepath.Clean(filepath.FromSlash(rawPath))
		if filepath.IsAbs(cleanRel) {
			problems = append(problems, fmt.Sprintf("%s must be relative, got %q", label, rawPath))
			continue
		}
		if cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
			problems = append(problems, fmt.Sprintf("%s must not escape artifact_root, got %q", label, rawPath))
			continue
		}
		absPath := filepath.Join(root, cleanRel)
		relToRoot, relErr := filepath.Rel(root, absPath)
		if relErr != nil {
			problems = append(problems, fmt.Sprintf("%s resolve document file %q: %v", label, rawPath, relErr))
			continue
		}
		if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			problems = append(problems, fmt.Sprintf("%s must not escape artifact_root, got %q", label, rawPath))
			continue
		}
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				problems = append(problems, fmt.Sprintf("%s references missing document file %q", label, rawPath))
				continue
			}
			problems = append(problems, fmt.Sprintf("%s stat document file %q: %v", label, rawPath, statErr))
			continue
		}
		if info.IsDir() {
			problems = append(problems, fmt.Sprintf("%s references a directory, not a document file: %q", label, rawPath))
			continue
		}
		raw, readErr := os.ReadFile(absPath)
		if readErr != nil {
			problems = append(problems, fmt.Sprintf("%s read document file %q: %v", label, rawPath, readErr))
			continue
		}
		if CollectDocumentBootstrapOnly(string(raw)) {
			problems = append(problems, fmt.Sprintf("%s references bootstrap-only collect document file %q", label, rawPath))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("shard pack manifest is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}
