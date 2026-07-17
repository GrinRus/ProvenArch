package providercommon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const collectManifestMissingFindingsRecoveryMode = "collect_manifest_missing_findings_recovery"

type collectManifestMissingFindingsRecoveryReport struct {
	BeforeDigest string
	AfterDigest  string
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
