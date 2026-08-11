package orchestrator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GrinRus/ProvenArch/internal/artifactaudit"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func (e *pipelineExecution) promoteValidatedArtifacts() error {
	if e.finalRunIndex == nil {
		return fmt.Errorf("promote validated artifacts: final run index is missing")
	}
	if e.validatorVerdict == nil {
		return fmt.Errorf("promote validated artifacts: validator verdict is missing")
	}
	if strings.TrimSpace(e.validatorVerdict.RunID) == "" && e.validatorVerdict.Verdict != "PASS" {
		return fmt.Errorf("promote validated artifacts: validator verdict is %s", e.validatorVerdict.Verdict)
	}
	auditReport, err := e.auditSelectedRunBeforePromotion()
	if err != nil {
		providerRaw, _ := e.workspace.ReadFile(runtimeValidatorVerdictPath(e.runID))
		if _, persistErr := persistEffectiveVerdict(e, *e.validatorVerdict, providerRaw, nil, auditReport); persistErr != nil {
			return persistErr
		}
		return err
	}
	providerRaw, _ := e.workspace.ReadFile(runtimeValidatorVerdictPath(e.runID))
	effective, err := persistEffectiveVerdict(e, *e.validatorVerdict, providerRaw, nil, auditReport)
	if err != nil {
		return err
	}
	if effective.Verdict != "PASS" {
		return fmt.Errorf("promote validated artifacts: effective verdict is %s", effective.Verdict)
	}

	generation, err := e.buildPromotionGeneration()
	if err != nil {
		return err
	}
	if err := e.validatePromotionGeneration(generation); err != nil {
		return err
	}
	staleArtifactPaths, err := e.collectStaleManagedCanonicalPaths(generation)
	if err != nil {
		return err
	}
	generation.staleArtifactPaths = staleArtifactPaths
	if err := e.activatePromotionGeneration(generation); err != nil {
		return err
	}

	e.removeArtifactsByPath(generation.staleArtifactPaths...)
	e.addArtifacts(generation.artifacts...)
	e.logInfo(e.stepStatus.CurrentStep, "", "validated artifacts promoted", map[string]any{
		"canonical_docs": len(e.finalRunIndex.CanonicalDocuments),
	})
	return nil
}

func (e *pipelineExecution) auditSelectedRunBeforePromotion() (artifactaudit.Report, error) {
	if !e.prePromotionAuditRequired {
		e.recordConformanceDiagnostic(map[string]any{"promotion_audit_result": "not_requested"})
		return artifactaudit.Report{Version: artifactaudit.Version, RunID: e.runID, Scope: "selected_run", Status: artifactaudit.StatusPass, Issues: []artifactaudit.Issue{}, Artifacts: []artifactaudit.Artifact{}, Summary: artifactaudit.Summary{}}, nil
	}
	report := artifactaudit.ScanSelectedRunWithCandidate(e.workspace, e.promotionRunID(), *e.validatorVerdict)
	if report.Status != artifactaudit.StatusFail {
		e.recordConformanceDiagnostic(map[string]any{"promotion_audit_result": string(report.Status)})
		return report, nil
	}
	e.recordConformanceDiagnostic(map[string]any{"promotion_audit_result": "fail", "validation_issue_class": "audit"})
	codes := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		if code := strings.TrimSpace(issue.Code); code != "" {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	if len(codes) > 5 {
		codes = codes[:5]
	}
	message := "selected-run audit failed"
	if len(codes) > 0 {
		message += ": " + strings.Join(codes, ", ")
	}
	e.logError(e.stepStatus.CurrentStep, "", message, map[string]any{
		"audit_status": report.Status,
		"issue_count":  len(report.Issues),
		"issue_codes":  codes,
	})
	return report, fmt.Errorf("promote validated artifacts: pre-promotion audit failed (%s)", strings.Join(codes, ", "))
}

type promotionGeneration struct {
	rootRel            string
	rootAbs            string
	journalRel         string
	journalAbs         string
	artifacts          []Artifact
	extraFilePaths     []string
	staleArtifactPaths []string
}

type promotionFaultPoint string

const (
	promotionFaultPrepareGeneration promotionFaultPoint = "prepare_generation"
	promotionFaultCopyDocument      promotionFaultPoint = "copy_document"
	promotionFaultBuildModel        promotionFaultPoint = "build_model"
	promotionFaultBuildDiagrams     promotionFaultPoint = "build_diagrams"
	promotionFaultBackupCanonical   promotionFaultPoint = "backup_canonical"
	promotionFaultActivateCanonical promotionFaultPoint = "activate_canonical"
)

type promotionFaultHook func(point promotionFaultPoint, relPath string) error

var promotionFaults struct {
	sync.Mutex
	hook promotionFaultHook
}

func setPromotionFaultHookForTest(hook promotionFaultHook) func() {
	promotionFaults.Lock()
	previous := promotionFaults.hook
	promotionFaults.hook = hook
	promotionFaults.Unlock()
	return func() {
		promotionFaults.Lock()
		promotionFaults.hook = previous
		promotionFaults.Unlock()
	}
}

func maybeFailPromotionOperation(point promotionFaultPoint, relPath string) error {
	promotionFaults.Lock()
	hook := promotionFaults.hook
	promotionFaults.Unlock()
	if hook == nil {
		return nil
	}
	return hook(point, relPath)
}

func (e *pipelineExecution) buildPromotionGeneration() (promotionGeneration, error) {
	runID := e.promotionRunID()
	baseRel := path.Join("reports", "taskruns", runID, "staging")
	generation := promotionGeneration{
		rootRel:    path.Join(baseRel, "promotion-generation"),
		journalRel: path.Join(baseRel, "promotion-journal"),
	}

	rootAbs, err := e.workspace.Resolve(generation.rootRel)
	if err != nil {
		return promotionGeneration{}, err
	}
	journalAbs, err := e.workspace.Resolve(generation.journalRel)
	if err != nil {
		return promotionGeneration{}, err
	}
	generation.rootAbs = rootAbs
	generation.journalAbs = journalAbs

	if err := maybeFailPromotionOperation(promotionFaultPrepareGeneration, generation.rootRel); err != nil {
		return promotionGeneration{}, err
	}
	if err := os.RemoveAll(generation.rootAbs); err != nil {
		return promotionGeneration{}, fmt.Errorf("reset promotion generation: %w", err)
	}
	if err := os.RemoveAll(generation.journalAbs); err != nil {
		return promotionGeneration{}, fmt.Errorf("reset promotion journal: %w", err)
	}
	if err := os.MkdirAll(generation.rootAbs, 0o755); err != nil {
		return promotionGeneration{}, fmt.Errorf("create promotion generation: %w", err)
	}

	generationRoot := workspace.Root{Path: generation.rootAbs}
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		if err := generationRootEnsureDir(generationRoot, prefix); err != nil {
			return promotionGeneration{}, err
		}
	}

	extraFileSet := map[string]struct{}{}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		canonicalPath := strings.TrimSpace(document.CanonicalPath)
		if canonicalPath == "" {
			return promotionGeneration{}, fmt.Errorf("promote validated artifacts: canonical path is empty")
		}
		if !isManagedCanonicalDocumentPath(canonicalPath) {
			return promotionGeneration{}, fmt.Errorf("promote validated artifacts: canonical path %q is not a managed generated artifact", canonicalPath)
		}
		content, err := e.workspace.ReadFile(document.StagedPath)
		if err != nil {
			return promotionGeneration{}, fmt.Errorf("read staged artifact %q: %w", document.StagedPath, err)
		}
		if err := maybeFailPromotionOperation(promotionFaultCopyDocument, canonicalPath); err != nil {
			return promotionGeneration{}, err
		}
		if err := generationRoot.WriteFile(canonicalPath, content); err != nil {
			return promotionGeneration{}, fmt.Errorf("stage promotion artifact %q: %w", canonicalPath, err)
		}
		generation.artifacts = append(generation.artifacts, Artifact{
			Path:  canonicalPath,
			Kind:  document.Kind,
			Label: document.Title,
		})
		if !isManagedCanonicalRootPath(canonicalPath) {
			extraFileSet[canonicalPath] = struct{}{}
		}
	}

	generationStore := model.NewStore(generationRoot)
	if err := maybeFailPromotionOperation(promotionFaultBuildModel, "model"); err != nil {
		return promotionGeneration{}, err
	}
	if err := rebuildDerivedModel(generationRoot, generationStore, e.finalRunIndex); err != nil {
		return promotionGeneration{}, err
	}
	entities, err := generationStore.ListEntities()
	if err != nil {
		return promotionGeneration{}, err
	}
	edges, err := generationStore.ListEdges()
	if err != nil {
		return promotionGeneration{}, err
	}
	if err := maybeFailPromotionOperation(promotionFaultBuildDiagrams, "reports/diagrams"); err != nil {
		return promotionGeneration{}, err
	}
	diagramArtifacts, err := reports.NewCompiler(generationRoot).CompileC4Diagrams(entities, edges)
	if err != nil {
		return promotionGeneration{}, err
	}
	generation.artifacts = append(generation.artifacts, toOrchestratorArtifacts(diagramArtifacts)...)

	generation.extraFilePaths = setKeysSorted(extraFileSet)
	return generation, nil
}

func (e *pipelineExecution) validatePromotionGeneration(generation promotionGeneration) error {
	generationRoot := workspace.Root{Path: generation.rootAbs}
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		abs, err := generationRoot.Resolve(prefix)
		if err != nil {
			return err
		}
		stat, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("validate promotion generation root %q: %w", prefix, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("validate promotion generation root %q: not a directory", prefix)
		}
	}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		canonicalPath := strings.TrimSpace(document.CanonicalPath)
		if !isManagedCanonicalDocumentPath(canonicalPath) {
			return fmt.Errorf("validate promotion generation: canonical path %q is unmanaged", canonicalPath)
		}
		abs, err := generationRoot.Resolve(canonicalPath)
		if err != nil {
			return err
		}
		stat, err := os.Stat(abs)
		if err != nil {
			return fmt.Errorf("validate promotion generation artifact %q: %w", canonicalPath, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("validate promotion generation artifact %q: expected file, got directory", canonicalPath)
		}
	}
	return nil
}

func (e *pipelineExecution) collectStaleManagedCanonicalPaths(generation promotionGeneration) ([]string, error) {
	expected := map[string]struct{}{}
	generationRoot := workspace.Root{Path: generation.rootAbs}
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		absRoot, err := generationRoot.Resolve(prefix)
		if err != nil {
			return nil, err
		}
		if err := filepath.WalkDir(absRoot, func(item string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(generation.rootAbs, item)
			if err != nil {
				return err
			}
			expected[filepath.ToSlash(relPath)] = struct{}{}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk promotion generation root %q: %w", prefix, err)
		}
	}

	stale := map[string]struct{}{}
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		absRoot, err := e.workspace.Resolve(prefix)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absRoot); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("inspect managed canonical surface %q: %w", prefix, err)
		}
		if err := filepath.WalkDir(absRoot, func(item string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relPath, err := filepath.Rel(e.workspace.Path, item)
			if err != nil {
				return err
			}
			canonicalPath := filepath.ToSlash(relPath)
			if _, ok := expected[canonicalPath]; !ok {
				stale[canonicalPath] = struct{}{}
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walk managed canonical surface %q: %w", prefix, err)
		}
	}
	return setKeysSorted(stale), nil
}

type promotionActivationRecord struct {
	prefix    string
	targetAbs string
	backupAbs string
	hadTarget bool
	activated bool
}

func (e *pipelineExecution) activatePromotionGeneration(generation promotionGeneration) error {
	if err := os.RemoveAll(generation.journalAbs); err != nil {
		return fmt.Errorf("reset promotion journal: %w", err)
	}
	if err := os.MkdirAll(generation.journalAbs, 0o755); err != nil {
		return fmt.Errorf("create promotion journal: %w", err)
	}

	records := []promotionActivationRecord{}
	rollback := func(cause error) error {
		if rollbackErr := rollbackPromotionActivation(records); rollbackErr != nil {
			return fmt.Errorf("%w; rollback failed: %v", cause, rollbackErr)
		}
		return cause
	}

	prefixes := managedCanonicalArtifactPrefixes()
	sort.Strings(prefixes)
	for _, prefix := range prefixes {
		sourceAbs := filepath.Join(generation.rootAbs, filepath.FromSlash(prefix))
		if err := ensurePromotionSourceDir(sourceAbs); err != nil {
			return rollback(fmt.Errorf("prepare promotion source %q: %w", prefix, err))
		}
		targetAbs, err := e.workspace.Resolve(prefix)
		if err != nil {
			return rollback(err)
		}
		backupAbs := filepath.Join(generation.journalAbs, filepath.FromSlash(prefix))
		record, err := activatePromotionPath(prefix, sourceAbs, targetAbs, backupAbs)
		if err != nil {
			if record.hadTarget || record.activated {
				records = append(records, record)
			}
			return rollback(err)
		}
		records = append(records, record)
	}

	for _, canonicalPath := range generation.extraFilePaths {
		sourceAbs := filepath.Join(generation.rootAbs, filepath.FromSlash(canonicalPath))
		targetAbs, err := e.workspace.Resolve(canonicalPath)
		if err != nil {
			return rollback(err)
		}
		backupAbs := filepath.Join(generation.journalAbs, filepath.FromSlash(canonicalPath))
		record, err := activatePromotionPath(canonicalPath, sourceAbs, targetAbs, backupAbs)
		if err != nil {
			if record.hadTarget || record.activated {
				records = append(records, record)
			}
			return rollback(err)
		}
		records = append(records, record)
	}

	_ = os.RemoveAll(generation.journalAbs)
	_ = os.RemoveAll(generation.rootAbs)
	return nil
}

func activatePromotionPath(prefix string, sourceAbs string, targetAbs string, backupAbs string) (promotionActivationRecord, error) {
	record := promotionActivationRecord{
		prefix:    prefix,
		targetAbs: targetAbs,
		backupAbs: backupAbs,
	}
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return record, fmt.Errorf("create canonical parent for %q: %w", prefix, err)
	}
	if err := os.MkdirAll(filepath.Dir(backupAbs), 0o755); err != nil {
		return record, fmt.Errorf("create promotion journal parent for %q: %w", prefix, err)
	}
	if _, err := os.Lstat(targetAbs); err == nil {
		if err := maybeFailPromotionOperation(promotionFaultBackupCanonical, prefix); err != nil {
			return record, err
		}
		if err := os.Rename(targetAbs, backupAbs); err != nil {
			return record, fmt.Errorf("journal canonical artifact %q: %w", prefix, err)
		}
		record.hadTarget = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return record, fmt.Errorf("inspect canonical artifact %q: %w", prefix, err)
	}
	if err := maybeFailPromotionOperation(promotionFaultActivateCanonical, prefix); err != nil {
		return record, err
	}
	if err := os.Rename(sourceAbs, targetAbs); err != nil {
		return record, fmt.Errorf("activate canonical artifact %q: %w", prefix, err)
	}
	record.activated = true
	return record, nil
}

func rollbackPromotionActivation(records []promotionActivationRecord) error {
	var rollbackErr error
	for idx := len(records) - 1; idx >= 0; idx-- {
		record := records[idx]
		if err := os.RemoveAll(record.targetAbs); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("remove partial canonical artifact %q: %w", record.prefix, err)
		}
		if !record.hadTarget {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(record.targetAbs), 0o755); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("recreate canonical parent for %q: %w", record.prefix, err)
			continue
		}
		if err := os.Rename(record.backupAbs, record.targetAbs); err != nil && rollbackErr == nil {
			rollbackErr = fmt.Errorf("restore canonical artifact %q: %w", record.prefix, err)
		}
	}
	return rollbackErr
}

func ensurePromotionSourceDir(abs string) error {
	stat, err := os.Stat(abs)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("not a directory")
		}
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(abs, 0o755)
	}
	return err
}

func generationRootEnsureDir(root workspace.Root, rel string) error {
	abs, err := root.Resolve(rel)
	if err != nil {
		return err
	}
	return os.MkdirAll(abs, 0o755)
}

func (e *pipelineExecution) promotionRunID() string {
	if trimmed := strings.TrimSpace(e.runID); trimmed != "" {
		return trimmed
	}
	if e.finalRunIndex != nil {
		if trimmed := strings.TrimSpace(e.finalRunIndex.RunID); trimmed != "" {
			return trimmed
		}
	}
	return "unknown-run"
}

func managedCanonicalArtifactPrefixes() []string {
	return []string{
		"model/edges",
		"model/entities",
		"reports/as-is",
		"reports/coverage",
		"reports/findings",
		"reports/agent-outputs",
		"reports/diagrams",
		"proposals",
	}
}

func isManagedCanonicalRootPath(canonicalPath string) bool {
	canonicalPath = strings.TrimSpace(path.Clean(filepath.ToSlash(canonicalPath)))
	for _, prefix := range managedCanonicalArtifactPrefixes() {
		if canonicalPathHasPrefix(canonicalPath, prefix) {
			return true
		}
	}
	return false
}

func isManagedCanonicalDocumentPath(canonicalPath string) bool {
	if isManagedCanonicalRootPath(canonicalPath) {
		return true
	}
	return canonicalPathHasPrefix(canonicalPath, "reports/changelog")
}

func canonicalPathHasPrefix(canonicalPath string, prefix string) bool {
	prefix = strings.TrimSpace(path.Clean(filepath.ToSlash(prefix)))
	return canonicalPath == prefix || strings.HasPrefix(canonicalPath, prefix+"/")
}

func rebuildDerivedModel(root workspace.Root, store model.Store, finalRunIndex *contracts.FinalRunIndex) error {
	if finalRunIndex == nil {
		return fmt.Errorf("rebuild derived model: final run index is missing")
	}
	for _, rel := range []string{"model/entities", "model/edges"} {
		abs, err := root.Resolve(rel)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("clear derived model dir %q: %w", rel, err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("recreate derived model dir %q: %w", rel, err)
		}
	}
	_, err := store.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: finalRunIndex.Semantic.Entities,
		Edges:    finalRunIndex.Semantic.Edges,
	})
	return err
}
