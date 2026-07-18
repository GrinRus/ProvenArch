package providercommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/qa"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
)

const (
	ShardPackManifestFileName = "shard-pack-manifest.json"
	ValidatorVerdictFileName  = "validator-verdict.json"
)

func ValidateRuntimeArtifacts(task acpruntime.Task, provider acpruntime.Provider) error {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep1Collect:
		return ValidateCollectArtifacts(task, provider)
	case acpruntime.StepProviderStep3Findings:
		return ValidateValidatorArtifacts(task)
	case acpruntime.StepProviderQA:
		return ValidateQAArtifacts(task)
	default:
		if runtimedrafts.IsDraftStep(task.StepID) {
			return ValidateDraftArtifacts(task)
		}
		return nil
	}
}

func ValidateCollectArtifacts(task acpruntime.Task, provider acpruntime.Provider) error {
	return artifactquality.ValidateCollectManifestInRootWithRepoRoots(task.WriteRoot, collectTaskRepoRoots(task))
}

func ValidateDraftArtifacts(task acpruntime.Task) error {
	_, _, err := ValidateRequiredRuntimeDraftArtifacts(task)
	return err
}

func ValidateRequiredRuntimeDraftArtifacts(task acpruntime.Task) (runtimedrafts.Manifest, []byte, error) {
	manifest, raw, err := runtimedrafts.ValidateRequiredManifest(
		task.WriteRoot,
		task.DraftFinalRoot,
		task.RunID,
		task.StepID,
		task.StepContract,
		task.ExpectedArtifacts,
	)
	if err != nil {
		return runtimedrafts.Manifest{}, nil, err
	}
	if strings.TrimSpace(task.StepID) == "init.step2.asis_docs" || strings.TrimSpace(task.StepID) == "refresh.step2.asis_docs" {
		if err := runtimedrafts.ValidateArchitectureHomeRepositoryReferences(task.DraftFinalRoot, manifest, collectTaskRepoRoots(task)); err != nil {
			return runtimedrafts.Manifest{}, nil, err
		}
	}
	return manifest, raw, nil
}

func ValidateValidatorArtifacts(task acpruntime.Task) error {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(task.WriteRoot), ValidatorVerdictFileName))
	if err != nil {
		return err
	}
	_, err = contracts.ParseValidatorVerdict(raw)
	return err
}

func ValidateQAArtifacts(task acpruntime.Task) error {
	answer, err := qa.ValidateAnswerFile(filepath.Join(filepath.Clean(task.WriteRoot), "qa-answer.json"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(answer.RunID) != strings.TrimSpace(task.RunID) {
		return fmt.Errorf("qa answer run_id %q does not match task run_id %q", answer.RunID, task.RunID)
	}
	if strings.TrimSpace(answer.Question) != strings.TrimSpace(task.Question) {
		return fmt.Errorf("qa answer question does not match task question")
	}
	contextPack, err := readQAContextPack(task.ContextPackPath)
	if err != nil {
		return err
	}
	return qa.ValidateAnswerAgainstContext(answer, contextPack)
}

func readQAContextPack(contextPackPath string) (qa.ContextPack, error) {
	contextPackPath = strings.TrimSpace(contextPackPath)
	if contextPackPath == "" {
		return qa.ContextPack{}, fmt.Errorf("qa context pack path is required")
	}
	raw, err := os.ReadFile(filepath.Clean(contextPackPath))
	if err != nil {
		return qa.ContextPack{}, fmt.Errorf("read qa context pack: %w", err)
	}
	return qa.ParseContextPack(raw)
}

func NonEmptyStage(stage string) string {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return "unknown"
	}
	return stage
}

func collectTaskRepoRoots(task acpruntime.Task) map[string]string {
	scopes := nonEmptyUniqueStrings(append([]string{task.RepoScope}, task.RepoScopes...))
	if len(scopes) == 0 {
		return nil
	}
	candidates := collectTaskRepoRootCandidates(task)
	if len(candidates) == 0 {
		return nil
	}
	roots := map[string]string{}
	for idx, scope := range scopes {
		if idx < len(candidates) {
			roots[scope] = candidates[idx]
		}
	}
	if len(scopes) == 1 && len(candidates) == 1 {
		roots[scopes[0]] = candidates[0]
	}
	return roots
}

func collectTaskRepoRootCandidates(task acpruntime.Task) []string {
	exclude := map[string]struct{}{}
	for _, value := range []string{task.Workspace, task.WriteRoot, task.DraftFinalRoot} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			exclude[filepath.Clean(trimmed)] = struct{}{}
		}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, root := range task.ReadContextRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		clean := filepath.Clean(root)
		if _, skip := exclude[clean]; skip {
			continue
		}
		info, err := os.Stat(clean)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func nonEmptyUniqueStrings(values []string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
