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
	return ValidateCollectManifestInRootWithRepoRoots(writeRoot, nil)
}

func ValidateCollectManifestInRootWithRepoRoots(writeRoot string, repoRoots map[string]string) error {
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
	if err := validateCollectManifestDocumentFiles(writeRoot, manifest.Documents); err != nil {
		return err
	}
	if err := validateCollectManifestRepoEvidencePaths(manifest, repoRoots); err != nil {
		return err
	}
	if CollectManifestSemanticBootstrapOnly(manifest) {
		return fmt.Errorf("shard pack manifest is invalid: semantic snapshot is bootstrap-only collect scaffold")
	}
	return nil
}

func ValidateCollectManifestBytes(raw []byte) error {
	_, err := contracts.ParseShardPackManifest(raw)
	return err
}

func ValidateCollectManifestTaskIdentity(
	manifest contracts.ShardPackManifest,
	runID string,
	stepID string,
	shardID string,
	domainID string,
	artifactRoot string,
) error {
	type identityField struct {
		name     string
		expected string
		actual   string
	}
	fields := []identityField{
		{name: "run_id", expected: runID, actual: manifest.RunID},
		{name: "step_id", expected: stepID, actual: manifest.StepID},
		{name: "shard_id", expected: shardID, actual: manifest.ShardID},
		{name: "domain_id", expected: domainID, actual: manifest.DomainID},
		{name: "artifact_root", expected: artifactRoot, actual: manifest.ArtifactRoot},
	}
	problems := make([]string, 0, len(fields))
	for _, field := range fields {
		expected := strings.TrimSpace(field.expected)
		if expected == "" {
			continue
		}
		actual := strings.TrimSpace(field.actual)
		if actual != expected {
			problems = append(problems, fmt.Sprintf("%s %q does not match task %s %q", field.name, actual, field.name, expected))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("shard pack manifest task identity is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
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
		text := string(raw)
		if CollectDocumentBootstrapOnly(text) {
			problems = append(problems, fmt.Sprintf("%s references bootstrap-only collect document file %q", label, rawPath))
			continue
		}
		if CollectDocumentRuntimeProcessContaminated(text) {
			problems = append(problems, fmt.Sprintf("%s references process-contaminated collect document file %q", label, rawPath))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("shard pack manifest is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateCollectManifestRepoEvidencePaths(manifest contracts.ShardPackManifest, repoRoots map[string]string) error {
	aliases := collectRepoRootAliases(repoRoots)
	if len(aliases) == 0 {
		return nil
	}
	problems := map[string]struct{}{}
	check := func(label string, repo string, evidencePath string) {
		repo = strings.TrimSpace(repo)
		evidencePath = strings.TrimSpace(evidencePath)
		if repo == "" && evidencePath == "" {
			return
		}
		if repo == "" {
			problems[label+".repo is required for repo evidence path"] = struct{}{}
			return
		}
		if evidencePath == "" {
			problems[label+".path is required for repo evidence"] = struct{}{}
			return
		}
		root := aliases[collectRepoRootAliasKey(repo)]
		if root == "" {
			problems[fmt.Sprintf("%s.repo %q has no resolved repo root", label, repo)] = struct{}{}
			return
		}
		if err := validateRepoEvidencePathExists(root, evidencePath); err != nil {
			problems[fmt.Sprintf("%s %v", label, err)] = struct{}{}
		}
	}
	for idx, citation := range manifest.Citations {
		check(fmt.Sprintf("citations[%d]", idx), citation.Repo, citation.Path)
	}
	for idx, entity := range manifest.Semantic.Entities {
		for evidenceIdx, evidence := range entity.Provenance.Evidence {
			check(fmt.Sprintf("semantic.entities[%d].provenance.evidence[%d]", idx, evidenceIdx), evidence.Repo, evidence.Path)
		}
	}
	for idx, edge := range manifest.Semantic.Edges {
		for evidenceIdx, evidence := range edge.Provenance.Evidence {
			check(fmt.Sprintf("semantic.edges[%d].provenance.evidence[%d]", idx, evidenceIdx), evidence.Repo, evidence.Path)
		}
	}
	for idx, finding := range manifest.Semantic.Findings {
		for evidenceIdx, evidence := range finding.Provenance.Evidence {
			check(fmt.Sprintf("semantic.findings[%d].provenance.evidence[%d]", idx, evidenceIdx), evidence.Repo, evidence.Path)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(problems))
	for problem := range problems {
		ordered = append(ordered, problem)
	}
	sort.Strings(ordered)
	return fmt.Errorf("shard pack manifest is invalid: %s", strings.Join(ordered, "; "))
}

func collectRepoRootAliases(repoRoots map[string]string) map[string]string {
	aliases := map[string]string{}
	register := func(alias string, root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		key := collectRepoRootAliasKey(alias)
		if key == "" {
			return
		}
		if _, exists := aliases[key]; !exists {
			aliases[key] = filepath.Clean(root)
		}
	}
	for scope, root := range repoRoots {
		register(scope, root)
		base := filepath.Base(filepath.Clean(strings.TrimSpace(root)))
		register(base, root)
		register(stripGeneratedRepoRootSuffix(base), root)
	}
	return aliases
}

func collectRepoRootAliasKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.Trim(value, "/")
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func stripGeneratedRepoRootSuffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	idx := strings.LastIndex(value, "-")
	if idx <= 0 || idx == len(value)-1 {
		return value
	}
	suffix := value[idx+1:]
	if len(suffix) < 8 {
		return value
	}
	for _, r := range suffix {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return value
		}
	}
	return value[:idx]
}

func validateRepoEvidencePathExists(repoRoot string, evidencePath string) error {
	root := filepath.Clean(strings.TrimSpace(repoRoot))
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(evidencePath)))
	if root == "" || root == "." {
		return fmt.Errorf("repo evidence root is empty")
	}
	if cleanRel == "" || cleanRel == "." {
		return fmt.Errorf("repo evidence path %q must not be empty", evidencePath)
	}
	if filepath.IsAbs(cleanRel) {
		return fmt.Errorf("repo evidence path %q must be relative", evidencePath)
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("repo evidence path %q must not escape repo root", evidencePath)
	}
	absPath := filepath.Join(root, cleanRel)
	relToRoot, relErr := filepath.Rel(root, absPath)
	if relErr != nil {
		return fmt.Errorf("resolve repo evidence path %q: %v", evidencePath, relErr)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return fmt.Errorf("repo evidence path %q must not escape repo root", evidencePath)
	}
	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("repo evidence path %q is a directory, not a file", evidencePath)
		}
		return nil
	}
	if filepath.Ext(cleanRel) == "" {
		if resolved, ok := resolveUniqueExtensionlessRepoEvidencePath(root, cleanRel); ok {
			if info, err := os.Stat(filepath.Join(root, resolved)); err == nil && !info.IsDir() {
				return nil
			}
		}
	}
	return fmt.Errorf("repo evidence path %q is missing under resolved repo root", evidencePath)
}

func resolveUniqueExtensionlessRepoEvidencePath(root string, cleanRel string) (string, bool) {
	parent := filepath.Join(root, filepath.Dir(cleanRel))
	entries, err := os.ReadDir(parent)
	if err != nil {
		return "", false
	}
	base := filepath.Base(cleanRel)
	matches := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, base+".") {
			matches = append(matches, name)
		}
	}
	if len(matches) != 1 {
		return "", false
	}
	return filepath.Join(filepath.Dir(cleanRel), matches[0]), true
}
