package providercommon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const (
	collectManifestMissingFindingsRecoveryMode = "collect_manifest_missing_findings_recovery"
	collectManifestProvenanceKindRecoveryMode  = "collect_manifest_provenance_kind_recovery"
	collectManifestTaskIdentityRecoveryMode    = "collect_manifest_task_identity_recovery"
)

type collectManifestMissingFindingsRecoveryReport struct {
	BeforeDigest string
	AfterDigest  string
}

type collectManifestProvenanceKindRecoveryReport struct {
	BeforeDigest     string
	AfterDigest      string
	ReplacementCount int
}

type collectManifestTaskIdentityRecoveryReport struct {
	BeforeDigest    string
	AfterDigest     string
	CorrectedFields []string
}

var collectManifestProvenanceKindAliases = map[string]string{
	"observed": "observation",
	"inferred": "inference",
	"asserted": "assertion",
}

func recoverCollectManifestTaskIdentity(task acpruntime.Task, validationErr error) (collectManifestTaskIdentityRecoveryReport, error) {
	if !classifyValidationIssues(validationErr).Has(issueCollectTaskIdentity) {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("collect manifest validation error is not eligible for task-identity recovery")
	}
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "." || strings.TrimSpace(task.WriteRoot) == "" {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("collect manifest task-identity recovery requires write_root")
	}
	manifestPath := filepath.Join(root, ShardPackManifestFileName)
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("read collect manifest: %w", err)
	}
	if err := artifactquality.ValidateCollectManifestInRootWithRepoRoots(root, collectTaskRepoRoots(task)); err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("collect manifest has a non-identity contract error: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(original, &manifest); err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("decode collect manifest: %w", err)
	}
	assigned := []struct {
		field string
		value string
	}{
		{field: "run_id", value: task.RunID},
		{field: "step_id", value: task.StepID},
		{field: "shard_id", value: task.ShardID},
		{field: "domain_id", value: task.DomainID},
		{field: "artifact_root", value: task.ArtifactRoot},
	}
	corrected := make([]string, 0, len(assigned))
	for _, identity := range assigned {
		expected := strings.TrimSpace(identity.value)
		if expected == "" {
			continue
		}
		actual, _ := manifest[identity.field].(string)
		if strings.TrimSpace(actual) == expected {
			continue
		}
		manifest[identity.field] = expected
		corrected = append(corrected, identity.field)
	}
	if len(corrected) == 0 {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("collect manifest has no mismatched assigned task identity fields")
	}
	sort.Strings(corrected)
	candidate, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("encode collect manifest: %w", err)
	}
	candidate = append(candidate, '\n')
	if err := artifactquality.ValidateCollectManifestBytes(candidate); err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("validate collect manifest candidate: %w", err)
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(manifestPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeCollectManifestAtomic(manifestPath, candidate, mode); err != nil {
		return collectManifestTaskIdentityRecoveryReport{}, err
	}
	if err := ValidateCollectArtifacts(task, ""); err != nil {
		if restoreErr := writeCollectManifestAtomic(manifestPath, original, mode); restoreErr != nil {
			return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("candidate validation failed (%v) and original manifest restore failed: %w", err, restoreErr)
		}
		return collectManifestTaskIdentityRecoveryReport{}, fmt.Errorf("validate completed collect manifest: %w", err)
	}
	return collectManifestTaskIdentityRecoveryReport{
		BeforeDigest:    collectManifestSHA256(original),
		AfterDigest:     collectManifestSHA256(candidate),
		CorrectedFields: corrected,
	}, nil
}

func recoverCollectManifestMissingFindings(task acpruntime.Task, validationErr error) (collectManifestMissingFindingsRecoveryReport, error) {
	if !isCollectManifestMissingFindingsOnlyCandidate(validationErr) {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("collect manifest validation error is not eligible for missing-findings recovery")
	}
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "." || strings.TrimSpace(task.WriteRoot) == "" {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("collect manifest missing-findings recovery requires write_root")
	}
	manifestPath := filepath.Join(root, ShardPackManifestFileName)
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("read collect manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(original, &manifest); err != nil {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("decode collect manifest: %w", err)
	}
	semantic, ok := manifest["semantic"].(map[string]any)
	if !ok {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("collect manifest semantic must be an object")
	}
	if _, exists := semantic["findings"]; exists {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("collect manifest semantic.findings already exists")
	}
	semantic["findings"] = []any{}
	candidate, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("encode collect manifest: %w", err)
	}
	candidate = append(candidate, '\n')
	if err := artifactquality.ValidateCollectManifestBytes(candidate); err != nil {
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("validate collect manifest candidate: %w", err)
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(manifestPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeCollectManifestAtomic(manifestPath, candidate, mode); err != nil {
		return collectManifestMissingFindingsRecoveryReport{}, err
	}
	if err := artifactquality.ValidateCollectManifestInRootWithRepoRoots(root, collectTaskRepoRoots(task)); err != nil {
		if restoreErr := writeCollectManifestAtomic(manifestPath, original, mode); restoreErr != nil {
			return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("candidate validation failed (%v) and original manifest restore failed: %w", err, restoreErr)
		}
		return collectManifestMissingFindingsRecoveryReport{}, fmt.Errorf("validate completed collect manifest: %w", err)
	}
	return collectManifestMissingFindingsRecoveryReport{
		BeforeDigest: collectManifestSHA256(original),
		AfterDigest:  collectManifestSHA256(candidate),
	}, nil
}

func isCollectManifestMissingFindingsOnlyCandidate(err error) bool {
	if err == nil {
		return false
	}
	detail := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(detail, "/semantic") &&
		strings.Contains(detail, "required: missing properties: 'findings'")
}

func recoverCollectManifestProvenanceKindAliases(task acpruntime.Task) (collectManifestProvenanceKindRecoveryReport, error) {
	root := filepath.Clean(strings.TrimSpace(task.WriteRoot))
	if root == "." || strings.TrimSpace(task.WriteRoot) == "" {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("collect manifest provenance-kind recovery requires write_root")
	}
	manifestPath := filepath.Join(root, ShardPackManifestFileName)
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("read collect manifest: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(original, &manifest); err != nil {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("decode collect manifest: %w", err)
	}
	semantic, ok := manifest["semantic"].(map[string]any)
	if !ok {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("collect manifest semantic must be an object")
	}
	replacementCount := 0
	for _, collectionName := range []string{"entities", "edges", "findings"} {
		collection, ok := semantic[collectionName].([]any)
		if !ok {
			continue
		}
		for _, rawItem := range collection {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			provenance, ok := item["provenance"].(map[string]any)
			if !ok {
				continue
			}
			kind, ok := provenance["kind"].(string)
			if !ok {
				continue
			}
			canonical, ok := collectManifestProvenanceKindAliases[kind]
			if !ok {
				continue
			}
			provenance["kind"] = canonical
			replacementCount++
		}
	}
	if replacementCount == 0 {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("collect manifest has no eligible provenance.kind aliases")
	}
	candidate, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("encode collect manifest: %w", err)
	}
	candidate = append(candidate, '\n')
	if err := artifactquality.ValidateCollectManifestBytes(candidate); err != nil {
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("validate collect manifest candidate: %w", err)
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(manifestPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeCollectManifestAtomic(manifestPath, candidate, mode); err != nil {
		return collectManifestProvenanceKindRecoveryReport{}, err
	}
	if err := artifactquality.ValidateCollectManifestInRootWithRepoRoots(root, collectTaskRepoRoots(task)); err != nil {
		if restoreErr := writeCollectManifestAtomic(manifestPath, original, mode); restoreErr != nil {
			return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("candidate validation failed (%v) and original manifest restore failed: %w", err, restoreErr)
		}
		return collectManifestProvenanceKindRecoveryReport{}, fmt.Errorf("validate completed collect manifest: %w", err)
	}
	return collectManifestProvenanceKindRecoveryReport{
		BeforeDigest:     collectManifestSHA256(original),
		AfterDigest:      collectManifestSHA256(candidate),
		ReplacementCount: replacementCount,
	}, nil
}

func writeCollectManifestAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create collect manifest temp file: %w", err)
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
		return fmt.Errorf("write collect manifest temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod collect manifest temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync collect manifest temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close collect manifest temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace collect manifest: %w", err)
	}
	removeTmp = false
	return nil
}

func collectManifestSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
