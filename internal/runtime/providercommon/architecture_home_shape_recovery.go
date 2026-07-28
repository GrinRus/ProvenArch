package providercommon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

const architectureHomeInlineHeadingRecoveryMode = "architecture_home_inline_heading_recovery"

type architectureHomeInlineHeadingRecoveryReport struct {
	BeforeDigest       string
	AfterDigest        string
	NormalizedSections []string
}

type architectureHomeUnsafeRestoreError struct {
	cause error
}

func (e architectureHomeUnsafeRestoreError) Error() string {
	return e.cause.Error()
}

func recoverArchitectureHomeInlineHeadings(
	task acpruntime.Task,
	validate func() error,
	validationErr error,
) (architectureHomeInlineHeadingRecoveryReport, bool, error) {
	if !isArchitectureHomeInlineHeadingOnlyCandidate(task, validationErr) {
		return architectureHomeInlineHeadingRecoveryReport{}, false, nil
	}
	manifest, _, err := runtimedrafts.Load(task.WriteRoot, runtimedrafts.AsIsManifestFile)
	if err != nil {
		return architectureHomeInlineHeadingRecoveryReport{}, true, fmt.Errorf("load Architecture Home manifest: %w", err)
	}
	overviewPath := ""
	for _, output := range manifest.Outputs {
		if filepath.ToSlash(path.Clean(strings.TrimSpace(output.CanonicalPath))) == "reports/as-is/overview.md" {
			if overviewPath != "" {
				return architectureHomeInlineHeadingRecoveryReport{}, true, fmt.Errorf("Architecture Home manifest has duplicate canonical overview outputs")
			}
			overviewPath = filepath.Join(filepath.Clean(task.DraftFinalRoot), filepath.Clean(output.Path))
		}
	}
	if overviewPath == "" {
		return architectureHomeInlineHeadingRecoveryReport{}, true, fmt.Errorf("Architecture Home manifest has no canonical overview output")
	}
	original, err := os.ReadFile(overviewPath)
	if err != nil {
		return architectureHomeInlineHeadingRecoveryReport{}, true, fmt.Errorf("read Architecture Home: %w", err)
	}
	candidate, sections, err := runtimedrafts.NormalizeArchitectureHomeInlineHeadings(original)
	if err != nil {
		return architectureHomeInlineHeadingRecoveryReport{}, true, err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(overviewPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := writeArchitectureHomeAtomic(overviewPath, candidate, mode); err != nil {
		return architectureHomeInlineHeadingRecoveryReport{}, true, err
	}
	if err := validate(); err != nil {
		if restoreErr := writeArchitectureHomeAtomic(overviewPath, original, mode); restoreErr != nil {
			return architectureHomeInlineHeadingRecoveryReport{}, true, architectureHomeUnsafeRestoreError{cause: fmt.Errorf("normalized Architecture Home validation failed (%v) and original restore failed: %w", err, restoreErr)}
		}
		return architectureHomeInlineHeadingRecoveryReport{}, true, fmt.Errorf("validate normalized Architecture Home: %w", err)
	}
	return architectureHomeInlineHeadingRecoveryReport{
		BeforeDigest:       architectureHomeSHA256(original),
		AfterDigest:        architectureHomeSHA256(candidate),
		NormalizedSections: sections,
	}, true, nil
}

func isArchitectureHomeInlineHeadingOnlyCandidate(task acpruntime.Task, err error) bool {
	stepID := strings.TrimSpace(task.StepID)
	if err == nil || (stepID != "init.step2.asis_docs" && stepID != "refresh.step2.asis_docs") {
		return false
	}
	detail := strings.TrimSpace(err.Error())
	marker := " Architecture Home has missing or empty required sections: "
	idx := strings.Index(detail, marker)
	if idx < 0 || strings.Contains(detail, "; ") || strings.Count(detail, "outputs[") != 1 {
		return false
	}
	want := strings.Join(runtimedrafts.ArchitectureHomeRequiredSections(), ", ")
	return strings.TrimSpace(detail[idx+len(marker):]) == want &&
		strings.HasPrefix(detail, "runtime draft manifest outputs are invalid: outputs[")
}

func writeArchitectureHomeAtomic(target string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create Architecture Home temp file: %w", err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write Architecture Home temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod Architecture Home temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync Architecture Home temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close Architecture Home temp file: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("replace Architecture Home: %w", err)
	}
	removeTemp = false
	return nil
}

func architectureHomeSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func markArchitectureHomeInlineHeadingsRecovered(result acpruntime.Result, report architectureHomeInlineHeadingRecoveryReport) acpruntime.Result {
	if result.Diagnostics == nil {
		result.Diagnostics = map[string]any{}
	}
	result.Diagnostics[architectureHomeInlineHeadingRecoveryMode] = map[string]any{
		"recovery_mode":            architectureHomeInlineHeadingRecoveryMode,
		"source":                   "runtime_shape_recovery",
		"provider_authored":        false,
		"normalized_sections":      append([]string(nil), report.NormalizedSections...),
		"before_digest":            report.BeforeDigest,
		"after_digest":             report.AfterDigest,
		"operator_review_required": true,
	}
	return result
}
