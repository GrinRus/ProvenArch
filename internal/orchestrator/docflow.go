package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/evidence"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	shardPackManifestFile = "shard-pack-manifest.json"
	finalRunIndexFile     = "final-run-index.json"
	citationIndexFile     = "citation-index.json"
	validatorVerdictFile  = "validator-verdict.json"
)

type aggregatedDocumentInfo struct {
	CanonicalID       string
	Kind              string
	Title             string
	Topics            map[string]struct{}
	CitationIDs       map[string]struct{}
	SourceShards      map[string]struct{}
	SourceDocumentIDs map[string]struct{}
}

type authoredDocumentAccumulator struct {
	Kind        string
	Title       string
	Topics      map[string]struct{}
	CitationIDs map[string]struct{}
	Content     []string
}

type stagedAuthoredDocument struct {
	CanonicalPath string
	Kind          string
	Title         string
	Content       string
}

func runtimeShardArtifactRoot(runID string, shardID string) string {
	return path.Join("reports", "taskruns", runID, "staging", "shards", sanitizeDomainArtifactSlug(shardID))
}

func runtimeFinalArtifactRoot(runID string) string {
	return path.Join("reports", "taskruns", runID, "staging", "final")
}

func runtimeValidatorArtifactRoot(runID string) string {
	return path.Join("reports", "taskruns", runID, "validator")
}

func runtimeFinalRunIndexPath(runID string) string {
	return path.Join(runtimeFinalArtifactRoot(runID), finalRunIndexFile)
}

func runtimeCitationIndexPath(runID string) string {
	return path.Join(runtimeFinalArtifactRoot(runID), citationIndexFile)
}

func runtimeValidatorVerdictPath(runID string) string {
	return path.Join(runtimeValidatorArtifactRoot(runID), validatorVerdictFile)
}

func runtimeAgentRole(stepID string) string {
	switch {
	case strings.HasSuffix(stepID, "step1.collect"):
		return "shard-analyst"
	case strings.HasSuffix(stepID, "step3.findings"):
		return "validator-findings"
	case acpruntime.IsQAStep(stepID):
		return "system-analyst-qa"
	default:
		return "runtime"
	}
}

func (e *pipelineExecution) runtimeArtifactContext(stepID string, shardID string, repoScopes []string) (string, string, string, []string, error) {
	var rel string
	switch {
	case strings.HasSuffix(stepID, "step1.collect"):
		rel = runtimeShardArtifactRoot(e.runID, shardID)
	case strings.HasSuffix(stepID, "step3.findings"):
		rel = runtimeValidatorArtifactRoot(e.runID)
	case acpruntime.IsQAStep(stepID):
		rel = runtimeQATaskRoot(e.runID)
	default:
		rel = runtimeStepWriteRoot(e.runID, stepID)
	}
	abs, err := e.workspace.Resolve(rel)
	if err != nil {
		return "", "", "", nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", "", "", nil, fmt.Errorf("create runtime artifact root: %w", err)
	}

	draftRel := runtimeDraftArtifactRoot(e.runID, stepID)
	draftAbs, err := e.workspace.Resolve(draftRel)
	if err != nil {
		return "", "", "", nil, err
	}
	if err := os.MkdirAll(draftAbs, 0o755); err != nil {
		return "", "", "", nil, fmt.Errorf("create runtime draft root: %w", err)
	}

	roots := []string{e.workspace.Path}
	if acpruntime.IsQAStep(stepID) {
		return rel, abs, abs, []string{abs}, nil
	}
	if strings.HasSuffix(stepID, "step3.findings") {
		if finalAbs, resolveErr := e.workspace.Resolve(runtimeFinalArtifactRoot(e.runID)); resolveErr == nil {
			roots = append(roots, finalAbs)
		}
	}
	if strings.HasSuffix(stepID, "step4.proposals") {
		if finalAbs, resolveErr := e.workspace.Resolve(runtimeFinalArtifactRoot(e.runID)); resolveErr == nil {
			roots = append(roots, finalAbs)
		}
	}
	for _, scope := range repoScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if repoPath := strings.TrimSpace(e.resolvedRepoPaths[scope]); repoPath != "" {
			roots = append(roots, repoPath)
		}
	}
	return rel, abs, draftAbs, normalizeOrderedUniqueStrings(roots), nil
}

func loadShardPackManifestFromRoot(root string) (contracts.ShardPackManifest, []byte, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(root), shardPackManifestFile))
	if err != nil {
		return contracts.ShardPackManifest{}, nil, fmt.Errorf("read shard pack manifest: %w", err)
	}
	manifest, err := contracts.ParseShardPackManifest(raw)
	if err != nil {
		return contracts.ShardPackManifest{}, nil, err
	}
	return manifest, raw, nil
}

func loadValidatorVerdictFromRoot(root string) (contracts.ValidatorVerdict, []byte, error) {
	raw, err := os.ReadFile(filepath.Join(filepath.Clean(root), validatorVerdictFile))
	if err != nil {
		return contracts.ValidatorVerdict{}, nil, fmt.Errorf("read validator verdict: %w", err)
	}
	verdict, err := contracts.ParseValidatorVerdict(raw)
	if err != nil {
		return contracts.ValidatorVerdict{}, nil, err
	}
	return verdict, raw, nil
}

type DocflowBuildInput struct {
	RunID             string
	Pipeline          Pipeline
	GeneratedAt       time.Time
	Workspace         workspace.Root
	ShardPacks        []contracts.ShardPackManifest
	ResolvedRepoPaths map[string]string
	Semantic          contracts.SemanticSnapshot
	RenderContext     reports.ReportRenderContext
	AsIsDraftManifest *runtimedrafts.Manifest
	AsIsDraftRoot     string
	DomainReports     func() (map[string]string, error)
	DomainEnvelopes   func() []reports.DomainTaskEnvelope
}

type DocflowBuildResult struct {
	StageArtifacts []Artifact
	CitationIndex  contracts.CitationIndex
	FinalRunIndex  contracts.FinalRunIndex
	Semantic       contracts.SemanticSnapshot
	EntitiesCount  int
	EdgesCount     int
}

func (e *pipelineExecution) assembleStagedDocFlow() error {
	repoAliases := newSemanticRepoAliasResolver(e.resolvedRepoPaths, e.shardPacks)
	evidencePaths := newSemanticEvidencePathResolver(e.resolvedRepoPaths, repoAliases)
	baseSemantic := aggregateSemanticSnapshot(e.shardPacks, repoAliases, evidencePaths)
	e.semanticBase = &baseSemantic
	result, err := buildStagedDocflow(DocflowBuildInput{
		RunID:             e.runID,
		Pipeline:          e.pipeline,
		GeneratedAt:       e.clock().UTC(),
		Workspace:         e.workspace,
		ShardPacks:        append([]contracts.ShardPackManifest(nil), e.shardPacks...),
		ResolvedRepoPaths: cloneStringMap(e.resolvedRepoPaths),
		Semantic:          e.effectiveSemanticSnapshot(),
		RenderContext:     e.renderContext(),
		AsIsDraftManifest: e.asIsDraftManifest,
		AsIsDraftRoot:     e.asIsDraftRoot,
		DomainReports:     e.authoredDomainReports,
		DomainEnvelopes:   e.stagedDomainEnvelopes,
	})
	if err != nil {
		return err
	}
	for _, artifact := range result.StageArtifacts {
		e.addArtifacts(artifact)
	}
	for _, document := range result.FinalRunIndex.CanonicalDocuments {
		e.addArtifacts(Artifact{
			Path:  document.StagedPath,
			Kind:  document.Kind,
			Label: document.Title,
		})
	}
	e.addArtifacts(Artifact{
		Path:  runtimeCitationIndexPath(e.runID),
		Kind:  "taskrun",
		Label: "Citation Index",
	})
	e.addArtifacts(Artifact{
		Path:  runtimeFinalRunIndexPath(e.runID),
		Kind:  "taskrun",
		Label: "Final Run Index",
	})
	e.finalRunIndex = &result.FinalRunIndex
	e.citationIndex = &result.CitationIndex
	if e.pipeline == PipelineRefresh {
		for _, warning := range assessRefreshArtifactWarnings(e.shardPacks, result.FinalRunIndex, result.CitationIndex) {
			e.addWarning(warning)
		}
	}
	e.findings = append([]contracts.Finding(nil), result.Semantic.Findings...)
	e.questions = append([]contracts.Question(nil), result.Semantic.Questions...)
	e.coverage = mergeCoverage(nil, &result.Semantic.Coverage)
	e.logInfo(e.stepStatus.CurrentStep, "", "staged doc flow assembled", map[string]any{
		"shard_packs":    len(e.shardPacks),
		"staged_docs":    len(result.FinalRunIndex.CanonicalDocuments),
		"citation_count": len(result.CitationIndex.Citations),
		"entities":       result.EntitiesCount,
		"edges":          result.EdgesCount,
		"findings":       len(result.Semantic.Findings),
		"questions":      len(result.Semantic.Questions),
	})
	return nil
}

func buildStagedDocflow(input DocflowBuildInput) (DocflowBuildResult, error) {
	stageRootRel := runtimeFinalArtifactRoot(input.RunID)
	stageRootAbs, err := input.Workspace.Resolve(stageRootRel)
	if err != nil {
		return DocflowBuildResult{}, err
	}
	if err := os.RemoveAll(stageRootAbs); err != nil {
		return DocflowBuildResult{}, fmt.Errorf("reset staged final root: %w", err)
	}
	if err := os.MkdirAll(stageRootAbs, 0o755); err != nil {
		return DocflowBuildResult{}, fmt.Errorf("create staged final root: %w", err)
	}
	stageRoot := workspace.Root{Path: stageRootAbs}

	repoAliases := newSemanticRepoAliasResolver(input.ResolvedRepoPaths, input.ShardPacks)
	evidencePaths := newSemanticEvidencePathResolver(input.ResolvedRepoPaths, repoAliases)
	semantic := normalizeSemanticSnapshot(input.Semantic, repoAliases, evidencePaths)
	documentInfos := aggregateDocumentInfos(input.ShardPacks)
	stageStore := model.NewStore(stageRoot)
	if _, err := stageStore.ApplySemanticSnapshot(contracts.SemanticSnapshot{
		Entities: semantic.Entities,
		Edges:    semantic.Edges,
	}); err != nil {
		return DocflowBuildResult{}, fmt.Errorf("apply staged semantic model: %w", err)
	}
	entities, err := stageStore.ListEntities()
	if err != nil {
		return DocflowBuildResult{}, err
	}
	edges, err := stageStore.ListEdges()
	if err != nil {
		return DocflowBuildResult{}, err
	}

	stageCompiler := reports.NewCompiler(stageRoot)
	renderCtx := input.RenderContext
	stageArtifactsByPath := map[string]Artifact{}
	canonicalPaths := map[string]struct{}{}

	registerStagedArtifact := func(artifact Artifact) {
		stagedPath := filepath.ToSlash(strings.TrimSpace(artifact.Path))
		if stagedPath == "" {
			return
		}
		artifact.Path = stagedPath
		if strings.TrimSpace(artifact.Kind) == "" {
			artifact.Kind = inferDocflowArtifactKind(stagedPath)
		}
		if strings.TrimSpace(artifact.Label) == "" {
			artifact.Label = path.Base(stripStagePrefix(stagedPath))
		}
		stageArtifactsByPath[stagedPath] = artifact
		canonicalPaths[stripStagePrefix(stagedPath)] = struct{}{}
	}
	registerCompiledArtifacts := func(items []reports.Artifact, itemErr error) error {
		if itemErr != nil {
			return itemErr
		}
		for _, artifact := range items {
			registerStagedArtifact(Artifact{
				Path:  path.Join(stageRootRel, artifact.Path),
				Kind:  artifact.Kind,
				Label: artifact.Label,
			})
		}
		return nil
	}

	authoredDocs, err := collectAuthoredStageDocuments(input.ShardPacks, input.Workspace.Path)
	if err != nil {
		return DocflowBuildResult{}, err
	}
	for _, document := range authoredDocs {
		content := document.Content
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := stageRoot.WriteFile(document.CanonicalPath, []byte(content)); err != nil {
			return DocflowBuildResult{}, err
		}
		registerStagedArtifact(Artifact{
			Path:  path.Join(stageRootRel, document.CanonicalPath),
			Kind:  document.Kind,
			Label: document.Title,
		})
	}
	if input.AsIsDraftManifest != nil {
		draftArtifacts, draftErr := applyRuntimeDraftOutputs(
			stageRoot,
			input.AsIsDraftRoot,
			*input.AsIsDraftManifest,
			stageRootRel,
			func(target string) bool {
				return strings.HasPrefix(target, "reports/as-is/") ||
					strings.HasPrefix(target, "reports/coverage/") ||
					strings.HasPrefix(target, "reports/agent-outputs/")
			},
		)
		if draftErr != nil {
			return DocflowBuildResult{}, draftErr
		}
		for _, artifact := range draftArtifacts {
			registerStagedArtifact(artifact)
		}
	}

	hasCanonicalPrefix := func(prefix string) bool {
		for canonicalPath := range canonicalPaths {
			if strings.HasPrefix(canonicalPath, prefix) {
				return true
			}
		}
		return false
	}
	hasCanonicalSuffix := func(suffix string) bool {
		for canonicalPath := range canonicalPaths {
			if strings.HasSuffix(canonicalPath, suffix) {
				return true
			}
		}
		return false
	}

	// Canonical narratives are runtime-authored only.
	// Compiler materializes derived technical artifacts and indexes, not fallback prose.
	if err := registerCompiledArtifacts(stageCompiler.WriteCoverage(&semantic.Coverage, semantic.Questions, renderCtx)); err != nil {
		return DocflowBuildResult{}, err
	}
	if err := registerCompiledArtifacts(stageCompiler.WriteFindings(semantic.Findings, renderCtx)); err != nil {
		return DocflowBuildResult{}, err
	}
	if !hasCanonicalPrefix("reports/agent-outputs/domains/") {
		if input.DomainReports == nil {
			return DocflowBuildResult{}, fmt.Errorf("domain report builder is not configured")
		}
		if domainReports, domainErr := input.DomainReports(); domainErr != nil {
			return DocflowBuildResult{}, domainErr
		} else if err := registerCompiledArtifacts(stageCompiler.WriteDomainOutputs(domainReports)); err != nil {
			return DocflowBuildResult{}, err
		}
	}
	if !hasCanonicalSuffix(".task-envelope.json") {
		if input.DomainEnvelopes == nil {
			return DocflowBuildResult{}, fmt.Errorf("domain envelope builder is not configured")
		}
		if err := registerCompiledArtifacts(stageCompiler.WriteDomainTaskEnvelopes(input.DomainEnvelopes())); err != nil {
			return DocflowBuildResult{}, err
		}
	}

	if err := registerCompiledArtifacts(stageCompiler.CompileC4Diagrams(entities, edges)); err != nil {
		return DocflowBuildResult{}, err
	}

	stageArtifactPaths := make([]string, 0, len(stageArtifactsByPath))
	for stagedPath := range stageArtifactsByPath {
		stageArtifactPaths = append(stageArtifactPaths, stagedPath)
	}
	sort.Strings(stageArtifactPaths)
	stageArtifacts := make([]Artifact, 0, len(stageArtifactPaths))
	for _, stagedPath := range stageArtifactPaths {
		stageArtifacts = append(stageArtifacts, stageArtifactsByPath[stagedPath])
	}

	citationIndex := aggregateCitationIndex(input.RunID, input.GeneratedAt, input.ShardPacks, documentInfos)
	citationRaw, err := json.MarshalIndent(citationIndex, "", "  ")
	if err != nil {
		return DocflowBuildResult{}, fmt.Errorf("marshal citation index: %w", err)
	}
	citationRaw = append(citationRaw, '\n')
	parsedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return DocflowBuildResult{}, err
	}

	finalRunIndex, err := buildFinalRunIndex(
		input.RunID,
		string(input.Pipeline),
		input.GeneratedAt,
		stageArtifacts,
		input.ShardPacks,
		documentInfos,
		parsedCitationIndex,
		semantic,
	)
	if err != nil {
		return DocflowBuildResult{}, err
	}
	parsedCitationIndex, _ = reconcileRuntimeDerivedCitationDocuments(parsedCitationIndex, finalRunIndex)
	citationRaw, err = json.MarshalIndent(parsedCitationIndex, "", "  ")
	if err != nil {
		return DocflowBuildResult{}, fmt.Errorf("marshal citation index: %w", err)
	}
	citationRaw = append(citationRaw, '\n')
	if err := stageRoot.WriteFile(citationIndexFile, citationRaw); err != nil {
		return DocflowBuildResult{}, err
	}
	parsedCitationIndex, err = contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return DocflowBuildResult{}, err
	}
	finalIndexRaw, err := json.MarshalIndent(finalRunIndex, "", "  ")
	if err != nil {
		return DocflowBuildResult{}, fmt.Errorf("marshal final run index: %w", err)
	}
	finalIndexRaw = append(finalIndexRaw, '\n')
	if err := stageRoot.WriteFile(finalRunIndexFile, finalIndexRaw); err != nil {
		return DocflowBuildResult{}, err
	}
	parsedFinalRunIndex, err := contracts.ParseFinalRunIndex(finalIndexRaw)
	if err != nil {
		return DocflowBuildResult{}, err
	}
	return DocflowBuildResult{
		StageArtifacts: stageArtifacts,
		CitationIndex:  parsedCitationIndex,
		FinalRunIndex:  parsedFinalRunIndex,
		Semantic:       semantic,
		EntitiesCount:  len(entities),
		EdgesCount:     len(edges),
	}, nil
}

func assessRefreshArtifactWarnings(
	manifests []contracts.ShardPackManifest,
	finalIndex contracts.FinalRunIndex,
	citationIndex contracts.CitationIndex,
) []string {
	warnings := []string{}

	if len(finalIndex.CanonicalDocuments) >= 2 && len(citationIndex.Citations) == 1 {
		onlyCitationID := strings.TrimSpace(citationIndex.Citations[0].ID)
		if artifactquality.IsGenericRuntimeSummaryCitation(onlyCitationID) {
			warnings = append(
				warnings,
				fmt.Sprintf(
					"artifact_quality: refresh staged final set has %d canonical documents but only 1 generic runtime-summary citation (%s)",
					len(finalIndex.CanonicalDocuments),
					onlyCitationID,
				),
			)
		}
	}

	if len(manifests) == 0 {
		return warnings
	}

	richManifestCount := 0
	for _, manifest := range manifests {
		if artifactquality.HasRepoSpecificCitationSurface(manifest) {
			richManifestCount++
		}
	}
	if richManifestCount == 0 {
		warnings = append(
			warnings,
			fmt.Sprintf(
				"artifact_quality: refresh collect manifests are reuse-only and preserve no repo-specific citations across %d shard(s)",
				len(manifests),
			),
		)
	}

	return warnings
}

func (e *pipelineExecution) authoredDomainReports() (map[string]string, error) {
	reportsByDomain := map[string]string{}
	for _, manifest := range e.shardPacks {
		for _, document := range manifest.Documents {
			if !strings.HasPrefix(strings.TrimSpace(document.CanonicalPath), "reports/agent-outputs/domains/") {
				continue
			}
			content, err := readShardDocument(manifest, document, e.workspace.Path)
			if err != nil {
				return nil, err
			}
			domainID := strings.TrimSuffix(path.Base(document.CanonicalPath), path.Ext(document.CanonicalPath))
			if existing := strings.TrimSpace(reportsByDomain[domainID]); existing != "" {
				reportsByDomain[domainID] = existing + "\n\n---\n\n" + strings.TrimSpace(content) + "\n"
				continue
			}
			reportsByDomain[domainID] = content
		}
	}
	return reportsByDomain, nil
}

func resolveManifestArtifactRoot(artifactRoot string, workspaceRoot string) (string, error) {
	artifactRoot = strings.TrimSpace(artifactRoot)
	if artifactRoot == "" {
		return "", fmt.Errorf("manifest artifact_root is empty")
	}
	if filepath.IsAbs(artifactRoot) {
		return filepath.Clean(artifactRoot), nil
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return "", fmt.Errorf("manifest artifact_root %q is relative but workspace root is empty", artifactRoot)
	}
	cleanWorkspaceRoot := filepath.Clean(workspaceRoot)
	resolved := filepath.Join(cleanWorkspaceRoot, filepath.Clean(filepath.FromSlash(artifactRoot)))
	relToWorkspace, err := filepath.Rel(cleanWorkspaceRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve relative artifact_root %q: %w", artifactRoot, err)
	}
	if relToWorkspace == ".." || strings.HasPrefix(relToWorkspace, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("manifest artifact_root %q escapes workspace root", artifactRoot)
	}
	return resolved, nil
}

func readShardDocument(manifest contracts.ShardPackManifest, document contracts.AuthoredDocument, workspaceRoot string) (string, error) {
	artifactRoot, err := resolveManifestArtifactRoot(manifest.ArtifactRoot, workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("read shard document %q: %w", document.ID, err)
	}
	relativePath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(document.Path)))
	if relativePath == "." || relativePath == "" {
		return "", fmt.Errorf("read shard document %q: manifest document path is empty", document.ID)
	}
	if filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("read shard document %q: manifest document path must be relative", document.ID)
	}
	cleanRoot := filepath.Clean(artifactRoot)
	absPath := filepath.Join(cleanRoot, relativePath)
	relToRoot, relErr := filepath.Rel(cleanRoot, absPath)
	if relErr != nil {
		return "", fmt.Errorf("read shard document %q: resolve artifact path: %w", document.ID, relErr)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("read shard document %q: manifest document path %q escapes artifact_root", document.ID, document.Path)
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("read shard document %q: %w", document.ID, err)
	}
	return string(content), nil
}

func collectAuthoredStageDocuments(manifests []contracts.ShardPackManifest, workspaceRoot string) ([]stagedAuthoredDocument, error) {
	documentsByCanonicalPath := map[string]*authoredDocumentAccumulator{}
	for _, manifest := range manifests {
		for _, document := range manifest.Documents {
			canonicalPath := filepath.ToSlash(strings.TrimSpace(document.CanonicalPath))
			if canonicalPath == "" {
				continue
			}
			accumulator, ok := documentsByCanonicalPath[canonicalPath]
			if !ok {
				accumulator = &authoredDocumentAccumulator{
					Kind:        strings.TrimSpace(document.Kind),
					Title:       strings.TrimSpace(document.Title),
					Topics:      map[string]struct{}{},
					CitationIDs: map[string]struct{}{},
					Content:     []string{},
				}
				documentsByCanonicalPath[canonicalPath] = accumulator
			}
			if strings.TrimSpace(accumulator.Kind) == "" {
				accumulator.Kind = strings.TrimSpace(document.Kind)
			}
			if strings.TrimSpace(accumulator.Title) == "" {
				accumulator.Title = strings.TrimSpace(document.Title)
			}
			for _, topic := range document.Topics {
				if trimmed := strings.TrimSpace(topic); trimmed != "" {
					accumulator.Topics[trimmed] = struct{}{}
				}
			}
			for _, citationID := range document.CitationIDs {
				if trimmed := strings.TrimSpace(citationID); trimmed != "" {
					accumulator.CitationIDs[trimmed] = struct{}{}
				}
			}
			content, err := readShardDocument(manifest, document, workspaceRoot)
			if err != nil {
				return nil, err
			}
			if trimmed := strings.TrimSpace(content); trimmed != "" {
				accumulator.Content = append(accumulator.Content, trimmed)
			}
		}
	}

	canonicalPaths := make([]string, 0, len(documentsByCanonicalPath))
	for canonicalPath := range documentsByCanonicalPath {
		canonicalPaths = append(canonicalPaths, canonicalPath)
	}
	sort.Strings(canonicalPaths)

	documents := make([]stagedAuthoredDocument, 0, len(canonicalPaths))
	for _, canonicalPath := range canonicalPaths {
		accumulator := documentsByCanonicalPath[canonicalPath]
		content := strings.Join(accumulator.Content, "\n\n---\n\n")
		if strings.TrimSpace(content) == "" {
			content = "# " + path.Base(canonicalPath) + "\n"
		}
		kind := strings.TrimSpace(accumulator.Kind)
		if kind == "" {
			kind = inferDocflowArtifactKind(canonicalPath)
		}
		title := strings.TrimSpace(accumulator.Title)
		if title == "" {
			title = path.Base(canonicalPath)
		}
		documents = append(documents, stagedAuthoredDocument{
			CanonicalPath: canonicalPath,
			Kind:          kind,
			Title:         title,
			Content:       content,
		})
	}
	return documents, nil
}

func inferDocflowArtifactKind(path string) string {
	canonicalPath := filepath.ToSlash(strings.TrimSpace(path))
	switch {
	case strings.HasSuffix(canonicalPath, ".mmd"):
		return "diagram"
	case strings.HasPrefix(canonicalPath, "proposals/"):
		return "proposal"
	case strings.HasSuffix(canonicalPath, ".task-envelope.json"):
		return "agent-output"
	case strings.HasSuffix(canonicalPath, ".json"):
		return "taskrun"
	default:
		return "report"
	}
}

func (e *pipelineExecution) stagedDomainEnvelopes() []reports.DomainTaskEnvelope {
	if len(e.domainRuns) == 0 {
		return nil
	}
	domainIDs := make([]string, 0, len(e.domainRuns))
	for domainID := range e.domainRuns {
		domainIDs = append(domainIDs, domainID)
	}
	sort.Strings(domainIDs)
	envelopes := make([]reports.DomainTaskEnvelope, 0, len(domainIDs))
	for _, domainID := range domainIDs {
		summary := e.domainRuns[domainID]
		envelopes = append(envelopes, reports.DomainTaskEnvelope{
			ContractVersion: 1,
			AgentID:         "domain-analyst",
			DomainID:        summary.DomainID,
			RepoScope:       summary.RepoScope,
			Unresolved:      append([]string(nil), summary.Unresolved...),
			Inputs: reports.DomainTaskInputs{
				DomainCardPath:      fmt.Sprintf("charter/cards/domains/%s.md", summary.DomainID),
				CoverageSummaryPath: "reports/coverage/summary.md",
				QuestionsPath:       "reports/coverage/open-questions.md",
				ModelEntitiesGlob:   "model/entities/*.yaml",
				FindingsPath:        "reports/findings/findings.md",
			},
			OutputPath: summary.OutputPath,
		})
	}
	return envelopes
}

type semanticRepoAliasResolver struct {
	exact map[string]string
	slug  map[string]string
}

type semanticEvidencePathResolver struct {
	rootsByRepo map[string]string
}

func newSemanticRepoAliasResolver(resolvedRepoPaths map[string]string, manifests []contracts.ShardPackManifest) semanticRepoAliasResolver {
	resolver := semanticRepoAliasResolver{
		exact: map[string]string{},
		slug:  map[string]string{},
	}
	for repoScope, repoPath := range resolvedRepoPaths {
		resolver.register(repoScope, repoScope)
		if base := strings.TrimSpace(filepath.Base(strings.TrimSpace(repoPath))); base != "" && base != "." {
			resolver.register(base, repoScope)
		}
	}
	for _, manifest := range manifests {
		for _, repoScope := range manifest.RepoScopes {
			resolver.register(repoScope, repoScope)
		}
	}
	return resolver
}

func (r semanticRepoAliasResolver) register(alias string, canonical string) {
	alias = strings.TrimSpace(alias)
	canonical = strings.TrimSpace(canonical)
	if alias == "" {
		return
	}
	if canonical == "" {
		canonical = alias
	}
	exactKey := strings.ToLower(alias)
	if _, exists := r.exact[exactKey]; !exists {
		r.exact[exactKey] = canonical
	}
	slugKey := slugutil.Slugify(stripGeneratedRepoSuffix(alias))
	if slugKey != "" {
		if _, exists := r.slug[slugKey]; !exists {
			r.slug[slugKey] = canonical
		}
	}
}

func (r semanticRepoAliasResolver) canonical(repo string) string {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return ""
	}
	if canonical, ok := r.exact[strings.ToLower(repo)]; ok {
		return canonical
	}
	if stripped := stripGeneratedRepoSuffix(repo); stripped != repo {
		if canonical, ok := r.exact[strings.ToLower(stripped)]; ok {
			return canonical
		}
	}
	if canonical, ok := r.slug[slugutil.Slugify(repo)]; ok {
		return canonical
	}
	if stripped := stripGeneratedRepoSuffix(repo); stripped != repo {
		if canonical, ok := r.slug[slugutil.Slugify(stripped)]; ok {
			return canonical
		}
	}
	return repo
}

func newSemanticEvidencePathResolver(resolvedRepoPaths map[string]string, repoAliases semanticRepoAliasResolver) semanticEvidencePathResolver {
	resolver := semanticEvidencePathResolver{rootsByRepo: map[string]string{}}
	for repoScope, repoPath := range resolvedRepoPaths {
		repoPath = strings.TrimSpace(repoPath)
		if repoPath == "" {
			continue
		}
		canonical := repoAliases.canonical(repoScope)
		if canonical == "" {
			canonical = strings.TrimSpace(repoScope)
		}
		if canonical == "" {
			continue
		}
		resolver.rootsByRepo[strings.ToLower(canonical)] = filepath.Clean(repoPath)
	}
	return resolver
}

func (r semanticEvidencePathResolver) resolve(repo string, evidencePath string) string {
	evidencePath = normalizeSemanticPath(evidencePath)
	if evidencePath == "" || filepath.IsAbs(evidencePath) || len(r.rootsByRepo) == 0 {
		return evidencePath
	}
	root := strings.TrimSpace(r.rootsByRepo[strings.ToLower(strings.TrimSpace(repo))])
	if root == "" {
		return evidencePath
	}
	return resolveUniqueExtensionlessEvidencePath(root, evidencePath)
}

func resolveUniqueExtensionlessEvidencePath(root string, evidencePath string) string {
	cleanRel := filepath.Clean(filepath.FromSlash(evidencePath))
	if cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
		return evidencePath
	}
	if filepath.Ext(cleanRel) != "" {
		return filepath.ToSlash(cleanRel)
	}
	exact := filepath.Join(root, cleanRel)
	if _, err := os.Stat(exact); err == nil {
		return filepath.ToSlash(cleanRel)
	}
	parent := filepath.Dir(exact)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return filepath.ToSlash(cleanRel)
	}
	base := filepath.Base(exact)
	matches := make([]string, 0, 1)
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
		return filepath.ToSlash(cleanRel)
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(cleanRel), matches[0]))
}

func aggregateSemanticSnapshot(manifests []contracts.ShardPackManifest, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver) contracts.SemanticSnapshot {
	snapshot := contracts.SemanticSnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	// Keep every shard observation until semantic normalization. Exact IDs can
	// legitimately repeat across shards when they describe the same logical
	// repository object; collapsing by map key here would silently discard
	// provenance and fields before the deterministic merge stage.
	entities := make([]contracts.Entity, 0)
	edges := make([]contracts.Edge, 0)
	findings := make([]contracts.Finding, 0)

	for _, manifest := range manifests {
		snapshot.Coverage = *mergeCoverage(&snapshot.Coverage, &manifest.Semantic.Coverage)
		snapshot.Questions = mergeQuestions(snapshot.Questions, manifest.Semantic.Questions)
		entities = append(entities, manifest.Semantic.Entities...)
		edges = append(edges, manifest.Semantic.Edges...)
		findings = append(findings, manifest.Semantic.Findings...)
	}

	snapshot.Entities = append(snapshot.Entities, entities...)
	snapshot.Edges = append(snapshot.Edges, edges...)
	snapshot.Findings = append(snapshot.Findings, findings...)
	sort.Slice(snapshot.Entities, func(i, j int) bool { return snapshot.Entities[i].ID < snapshot.Entities[j].ID })
	sort.Slice(snapshot.Edges, func(i, j int) bool { return snapshot.Edges[i].ID < snapshot.Edges[j].ID })
	sort.Slice(snapshot.Findings, func(i, j int) bool { return snapshot.Findings[i].ID < snapshot.Findings[j].ID })
	return normalizeSemanticSnapshot(snapshot, repoAliases, evidencePaths)
}

func normalizeSemanticSnapshot(snapshot contracts.SemanticSnapshot, repoAliases semanticRepoAliasResolver, evidenceResolvers ...semanticEvidencePathResolver) contracts.SemanticSnapshot {
	evidencePaths := semanticEvidencePathResolver{}
	if len(evidenceResolvers) > 0 {
		evidencePaths = evidenceResolvers[0]
	}
	snapshot.Coverage.Observed = dedupeSemanticStrings(snapshot.Coverage.Observed)
	snapshot.Coverage.Missing = dedupeSemanticStrings(canonicalizeCoverageMissing(snapshot.Coverage.Missing))
	snapshot.Coverage.Notes = dedupeSemanticStrings(snapshot.Coverage.Notes)

	entities, entityRemap := dedupeSemanticEntities(snapshot.Entities, repoAliases, evidencePaths)
	snapshot.Entities = entities
	endpointRemap := newSemanticEndpointRemap(entities, entityRemap)
	snapshot.Edges = dedupeSemanticEdges(snapshot.Edges, repoAliases, evidencePaths, endpointRemap)
	snapshot.Findings = dedupeSemanticFindings(snapshot.Findings, repoAliases, evidencePaths, entityRemap)
	snapshot.Questions = mergeQuestions(nil, rewriteSemanticQuestions(snapshot.Questions, entityRemap))
	return snapshot
}

func dedupeSemanticEntities(entities []contracts.Entity, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver) ([]contracts.Entity, map[string]string) {
	normalizedEntities := make([]contracts.Entity, 0, len(entities))
	exactObservationCounts := map[string]int{}
	for _, entity := range entities {
		entity = normalizeSemanticEntity(entity, repoAliases, evidencePaths)
		normalizedEntities = append(normalizedEntities, entity)
		if id := normalizeSemanticKey(entity.ID); id != "" {
			repo := normalizeSemanticKey(primarySemanticEvidenceRepo(entity.Provenance.Evidence))
			typeKey := normalizeSemanticKey(entity.Type)
			if repo != "" {
				exactObservationCounts[strings.Join([]string{id, typeKey, repo}, "|")]++
			}
		}
	}
	normalizedGroups := map[string][]contracts.Entity{}
	order := []string{}
	for _, entity := range normalizedEntities {
		key := semanticEntityDedupKey(entity)
		if id := normalizeSemanticKey(entity.ID); id != "" {
			repo := normalizeSemanticKey(primarySemanticEvidenceRepo(entity.Provenance.Evidence))
			typeKey := normalizeSemanticKey(entity.Type)
			if repo != "" && exactObservationCounts[strings.Join([]string{id, typeKey, repo}, "|")] > 1 {
				// An exact ID repeated by the same logical repository is an
				// explicit identity claim; merge its observations after the
				// collision compatibility check, preserving all evidence.
				key = strings.Join([]string{"id", id, typeKey, repo}, "|")
			}
		}
		if _, exists := normalizedGroups[key]; !exists {
			order = append(order, key)
		}
		normalizedGroups[key] = append(normalizedGroups[key], entity)
	}

	mergedEntities := make([]contracts.Entity, 0, len(order))
	remap := map[string]string{}
	for _, key := range order {
		group := normalizedGroups[key]
		if len(group) == 0 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return canonicalSemanticIDSortKey(group[i].ID) < canonicalSemanticIDSortKey(group[j].ID)
		})
		winner := group[0]
		winner.Aliases = dedupeExactStrings(winner.Aliases)
		winner.Tags = dedupeSemanticStrings(winner.Tags)
		for _, candidate := range group[1:] {
			remap[strings.TrimSpace(candidate.ID)] = strings.TrimSpace(winner.ID)
			winner = mergeSemanticEntity(winner, candidate)
		}
		mergedEntities = append(mergedEntities, winner)
	}
	sort.Slice(mergedEntities, func(i, j int) bool { return mergedEntities[i].ID < mergedEntities[j].ID })
	return mergedEntities, remap
}

type semanticEndpointRemap struct {
	exact map[string]string
	token map[string]string
}

func newSemanticEndpointRemap(entities []contracts.Entity, entityRemap map[string]string) semanticEndpointRemap {
	resolver := semanticEndpointRemap{
		exact: map[string]string{},
		token: map[string]string{},
	}
	ambiguousExact := map[string]struct{}{}
	ambiguousToken := map[string]struct{}{}
	registerUnique := func(values map[string]string, ambiguous map[string]struct{}, key string, canonicalID string) {
		key = strings.TrimSpace(key)
		canonicalID = strings.TrimSpace(canonicalID)
		if key == "" || canonicalID == "" {
			return
		}
		if _, blocked := ambiguous[key]; blocked {
			return
		}
		if existing, exists := values[key]; exists && existing != canonicalID {
			delete(values, key)
			ambiguous[key] = struct{}{}
			return
		}
		values[key] = canonicalID
	}
	for candidateID, canonicalID := range entityRemap {
		registerUnique(resolver.exact, ambiguousExact, candidateID, canonicalID)
	}
	for _, entity := range entities {
		canonicalID := strings.TrimSpace(entity.ID)
		registerUnique(resolver.exact, ambiguousExact, canonicalID, canonicalID)
		for _, alias := range entity.Aliases {
			registerUnique(resolver.exact, ambiguousExact, alias, canonicalID)
		}
		for _, value := range append([]string{canonicalID, entity.Name}, entity.Aliases...) {
			registerUnique(resolver.token, ambiguousToken, semanticEntityIdentityToken(value), canonicalID)
		}
	}
	return resolver
}

func normalizeSemanticEntity(entity contracts.Entity, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver) contracts.Entity {
	entity.ID = strings.TrimSpace(entity.ID)
	entity.Type = normalizeSemanticEntityType(entity.Type)
	entity.Name = strings.TrimSpace(entity.Name)
	entity.OwnerTeamID = strings.TrimSpace(entity.OwnerTeamID)
	entity.Aliases = dedupeExactStrings(entity.Aliases)
	entity.Tags = dedupeSemanticStrings(entity.Tags)
	entity.Provenance = normalizeSemanticProvenance(entity.Provenance, repoAliases, evidencePaths)
	return entity
}

func mergeSemanticEntity(winner contracts.Entity, candidate contracts.Entity) contracts.Entity {
	winner.Aliases = dedupeExactStrings(append(append([]string{}, winner.Aliases...), candidate.Aliases...))
	winner.Aliases = appendSemanticAlias(winner.Aliases, candidate.ID, winner.ID)
	winner.Tags = dedupeSemanticStrings(append(winner.Tags, candidate.Tags...))
	if strings.TrimSpace(winner.Name) == "" {
		winner.Name = strings.TrimSpace(candidate.Name)
	}
	if strings.TrimSpace(winner.OwnerTeamID) == "" {
		winner.OwnerTeamID = strings.TrimSpace(candidate.OwnerTeamID)
	}
	if winner.Attributes == nil && candidate.Attributes != nil {
		winner.Attributes = candidate.Attributes
	}
	winner.Provenance = mergeSemanticProvenance(winner.Provenance, candidate.Provenance)
	return winner
}

func dedupeSemanticEdges(edges []contracts.Edge, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver, endpointRemap semanticEndpointRemap) []contracts.Edge {
	normalizedEdges := make([]contracts.Edge, 0, len(edges))
	exactIDSignatures := map[string]map[string]struct{}{}
	exactIDCounts := map[string]int{}
	for _, edge := range edges {
		edge.ID = strings.TrimSpace(edge.ID)
		edge.Type = strings.TrimSpace(edge.Type)
		edge.Name = strings.TrimSpace(edge.Name)
		edge.From = rewriteSemanticEndpointID(edge.From, endpointRemap)
		edge.To = rewriteSemanticEndpointID(edge.To, endpointRemap)
		edge.Provenance = normalizeSemanticProvenance(edge.Provenance, repoAliases, evidencePaths)
		normalizedEdges = append(normalizedEdges, edge)
		id := normalizeSemanticKey(edge.ID)
		if id == "" {
			continue
		}
		signature := strings.Join([]string{normalizeSemanticKey(edge.Type), strings.TrimSpace(edge.From), strings.TrimSpace(edge.To)}, "\x00")
		if exactIDSignatures[id] == nil {
			exactIDSignatures[id] = map[string]struct{}{}
		}
		exactIDSignatures[id][signature] = struct{}{}
		exactIDCounts[id]++
	}
	grouped := map[string][]contracts.Edge{}
	order := []string{}
	for _, edge := range normalizedEdges {
		id := normalizeSemanticKey(edge.ID)
		if exactIDCounts[id] > 1 {
			if len(exactIDSignatures[id]) > 1 {
				// The provider reused a weak edge ID for distinct endpoint
				// pairs. Canonicalize each pair before final identity checks.
				edge.ID = canonicalSemanticEdgeID(edge)
			} else {
				// Same exact edge identity observed with different evidence;
				// merge it and retain all provenance.
				key := "exact|" + id
				if _, exists := grouped[key]; !exists {
					order = append(order, key)
				}
				grouped[key] = append(grouped[key], edge)
				continue
			}
		}
		key := semanticEdgeDedupKey(edge)
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], edge)
	}

	merged := make([]contracts.Edge, 0, len(order))
	for _, key := range order {
		group := grouped[key]
		if len(group) == 0 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return canonicalSemanticIDSortKey(group[i].ID) < canonicalSemanticIDSortKey(group[j].ID)
		})
		winner := group[0]
		for _, candidate := range group[1:] {
			winner.Provenance = mergeSemanticProvenance(winner.Provenance, candidate.Provenance)
			if strings.TrimSpace(winner.Name) == "" {
				winner.Name = strings.TrimSpace(candidate.Name)
			}
		}
		merged = append(merged, winner)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].ID < merged[j].ID })
	return merged
}

func dedupeSemanticFindings(findings []contracts.Finding, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver, entityRemap map[string]string) []contracts.Finding {
	grouped := map[string][]contracts.Finding{}
	order := []string{}
	for _, finding := range findings {
		finding.ID = strings.TrimSpace(finding.ID)
		finding.Title = strings.TrimSpace(finding.Title)
		finding.Description = strings.TrimSpace(finding.Description)
		finding.Severity = strings.TrimSpace(finding.Severity)
		finding.RuleID = strings.TrimSpace(finding.RuleID)
		finding.RelatedIDs = rewriteSemanticRelatedIDs(finding.RelatedIDs, entityRemap)
		finding.Provenance = normalizeSemanticProvenance(finding.Provenance, repoAliases, evidencePaths)
		key := semanticFindingDedupKey(finding)
		if _, exists := grouped[key]; !exists {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], finding)
	}
	merged := make([]contracts.Finding, 0, len(order))
	for _, key := range order {
		group := grouped[key]
		if len(group) == 0 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return canonicalSemanticIDSortKey(group[i].ID) < canonicalSemanticIDSortKey(group[j].ID)
		})
		winner := group[0]
		winner.RelatedIDs = dedupeSemanticStrings(winner.RelatedIDs)
		for _, candidate := range group[1:] {
			winner = mergeSemanticFinding(winner, candidate)
		}
		merged = append(merged, winner)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].ID < merged[j].ID })
	return merged
}

func mergeSemanticFinding(winner contracts.Finding, candidate contracts.Finding) contracts.Finding {
	if strings.TrimSpace(winner.ID) == "" {
		winner.ID = strings.TrimSpace(candidate.ID)
	}
	if strings.TrimSpace(winner.Severity) == "" {
		winner.Severity = strings.TrimSpace(candidate.Severity)
	}
	if strings.TrimSpace(winner.Title) == "" {
		winner.Title = strings.TrimSpace(candidate.Title)
	}
	if strings.TrimSpace(winner.Description) == "" {
		winner.Description = strings.TrimSpace(candidate.Description)
	}
	if strings.TrimSpace(winner.RuleID) == "" {
		winner.RuleID = strings.TrimSpace(candidate.RuleID)
	}
	winner.RelatedIDs = dedupeSemanticStrings(append(append([]string{}, winner.RelatedIDs...), candidate.RelatedIDs...))
	winner.Provenance = mergeSemanticProvenance(winner.Provenance, candidate.Provenance)
	return winner
}

func rewriteSemanticQuestions(questions []contracts.Question, entityRemap map[string]string) []contracts.Question {
	rewritten := make([]contracts.Question, 0, len(questions))
	for _, question := range questions {
		question.ID = strings.TrimSpace(question.ID)
		question.Text = strings.TrimSpace(question.Text)
		question.Priority = strings.TrimSpace(question.Priority)
		question.RelatedIDs = rewriteSemanticRelatedIDs(question.RelatedIDs, entityRemap)
		rewritten = append(rewritten, question)
	}
	return rewritten
}

func rewriteSemanticRelatedIDs(values []string, entityRemap map[string]string) []string {
	rewritten := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		canonical := rewriteSemanticID(value, entityRemap)
		if canonical == "" {
			continue
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		rewritten = append(rewritten, canonical)
	}
	sort.Strings(rewritten)
	return rewritten
}

func rewriteSemanticID(value string, entityRemap map[string]string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if remapped, ok := entityRemap[value]; ok && strings.TrimSpace(remapped) != "" {
		return strings.TrimSpace(remapped)
	}
	return value
}

func rewriteSemanticEndpointID(value string, endpointRemap semanticEndpointRemap) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if remapped, ok := endpointRemap.exact[value]; ok && strings.TrimSpace(remapped) != "" {
		return strings.TrimSpace(remapped)
	}
	token := semanticEntityIdentityToken(value)
	if remapped, ok := endpointRemap.token[token]; ok && strings.TrimSpace(remapped) != "" {
		return strings.TrimSpace(remapped)
	}
	return value
}

func normalizeSemanticProvenance(provenance contracts.Provenance, repoAliases semanticRepoAliasResolver, evidencePaths semanticEvidencePathResolver) contracts.Provenance {
	provenance.Kind = strings.TrimSpace(provenance.Kind)
	provenance.Evidence = normalizeSemanticEvidenceSet(provenance.Evidence, repoAliases, evidencePaths)
	return provenance
}

func normalizeSemanticEvidenceSet(evidence []contracts.Evidence, repoAliases semanticRepoAliasResolver, evidenceResolvers ...semanticEvidencePathResolver) []contracts.Evidence {
	evidencePaths := semanticEvidencePathResolver{}
	if len(evidenceResolvers) > 0 {
		evidencePaths = evidenceResolvers[0]
	}
	type keyedEvidence struct {
		key      string
		evidence contracts.Evidence
	}
	items := make([]keyedEvidence, 0, len(evidence))
	seen := map[string]struct{}{}
	for _, item := range evidence {
		item.Repo = repoAliases.canonical(item.Repo)
		item.Path = normalizeSemanticPath(item.Path)
		item.Path = evidencePaths.resolve(item.Repo, item.Path)
		item.Ref = strings.TrimSpace(item.Ref)
		item.ExcerptHash = strings.TrimSpace(item.ExcerptHash)
		item.Excerpt = strings.TrimSpace(item.Excerpt)
		key := semanticEvidenceKey(item)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, keyedEvidence{key: key, evidence: item})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })
	normalized := make([]contracts.Evidence, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, item.evidence)
	}
	return normalized
}

func mergeSemanticProvenance(winner contracts.Provenance, candidate contracts.Provenance) contracts.Provenance {
	if strings.TrimSpace(winner.Kind) == "" {
		winner.Kind = strings.TrimSpace(candidate.Kind)
	}
	if strings.EqualFold(strings.TrimSpace(candidate.Kind), "observation") && !strings.EqualFold(strings.TrimSpace(winner.Kind), "observation") {
		winner.Kind = strings.TrimSpace(candidate.Kind)
	}
	if candidate.Confidence > winner.Confidence {
		winner.Confidence = candidate.Confidence
	}
	winner.Evidence = normalizeSemanticEvidenceSet(append(append([]contracts.Evidence{}, winner.Evidence...), candidate.Evidence...), semanticRepoAliasResolver{})
	return winner
}

func semanticEntityDedupKey(entity contracts.Entity) string {
	entityType := normalizeSemanticKey(entity.Type)
	repo := normalizeSemanticKey(primarySemanticEvidenceRepo(entity.Provenance.Evidence))
	name := normalizeSemanticEntityNameDedupKey(entity.Type, entity.Name)
	idToken := semanticEntityIdentityToken(entity.ID)
	nameToken := semanticEntityIdentityToken(entity.Name)
	if entityType != "" && repo != "" && idToken != "" && idToken == nameToken {
		return strings.Join([]string{"identity", entityType, repo, semanticEntityIDIdentityToken(entity.ID)}, "|")
	}
	evidencePath := normalizeSemanticKey(primarySemanticEvidencePath(entity.Provenance.Evidence))
	if name == "" || (repo == "" && evidencePath == "") {
		return "id|" + strings.TrimSpace(entity.ID)
	}
	return strings.Join([]string{entityType, repo, name, evidencePath}, "|")
}

func normalizeSemanticEntityType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "database", "data-store", "data store":
		return "datastore"
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeSemanticEntityNameDedupKey(entityType string, name string) string {
	normalizedName := normalizeSemanticKey(name)
	if normalizedName == "" {
		return ""
	}
	if strings.Contains(normalizeSemanticKey(entityType), "service") {
		collapsed := strings.ReplaceAll(normalizedName, " ", "")
		if collapsed != "" {
			return collapsed
		}
	}
	return normalizedName
}

func semanticEntityIdentityToken(value string) string {
	value = strings.TrimSpace(value)
	if splitAt := strings.LastIndexAny(value, ".:/\\"); splitAt >= 0 {
		value = value[splitAt+1:]
	}
	return strings.ReplaceAll(normalizeSemanticKey(value), " ", "")
}

func semanticEntityIDIdentityToken(value string) string {
	value = strings.NewReplacer(".", " ", ":", " ", "/", " ", "\\", " ").Replace(strings.TrimSpace(value))
	return strings.ReplaceAll(normalizeSemanticKey(value), " ", "")
}

func semanticFindingDedupKey(finding contracts.Finding) string {
	ruleID := normalizeSemanticKey(finding.RuleID)
	title := normalizeSemanticKey(finding.Title)
	related := normalizedSemanticRelatedIDSet(finding.RelatedIDs)
	if ruleID == "" && title == "" && len(related) == 0 {
		return "id|" + strings.TrimSpace(finding.ID)
	}
	return "sig|" + strings.Join([]string{ruleID, title, strings.Join(related, ",")}, "|")
}

func normalizedSemanticRelatedIDSet(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		key := normalizeSemanticKey(value)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	return normalized
}

func semanticEdgeDedupKey(edge contracts.Edge) string {
	edgeType := normalizeSemanticKey(edge.Type)
	fromID := strings.TrimSpace(edge.From)
	toID := strings.TrimSpace(edge.To)
	evidencePath := normalizeSemanticKey(primarySemanticEvidencePath(edge.Provenance.Evidence))
	if fromID == "" || toID == "" {
		return "id|" + strings.TrimSpace(edge.ID)
	}
	return strings.Join([]string{edgeType, fromID, toID, evidencePath}, "|")
}

func canonicalSemanticEdgeID(edge contracts.Edge) string {
	return strings.Join([]string{
		"edge",
		canonicalSemanticIDPart(edge.From),
		canonicalSemanticIDPart(edge.Type),
		canonicalSemanticIDPart(edge.To),
	}, ".")
}

func canonicalSemanticIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	previousSeparator := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' {
			b.WriteRune(r)
			previousSeparator = false
			continue
		}
		if !previousSeparator {
			b.WriteByte('-')
			previousSeparator = true
		}
	}
	return strings.Trim(b.String(), "-._")
}

func primarySemanticEvidenceRepo(evidence []contracts.Evidence) string {
	if len(evidence) == 0 {
		return ""
	}
	return strings.TrimSpace(evidence[0].Repo)
}

func primarySemanticEvidencePath(evidence []contracts.Evidence) string {
	if len(evidence) == 0 {
		return ""
	}
	return normalizeSemanticPath(evidence[0].Path)
}

func normalizeSemanticPath(value string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
	if clean == "." {
		return ""
	}
	return clean
}

func semanticEvidenceKey(evidence contracts.Evidence) string {
	lines := ""
	if evidence.Lines != nil {
		lines = fmt.Sprintf("%d:%d", evidence.Lines.Start, evidence.Lines.End)
	}
	return strings.Join([]string{
		strings.TrimSpace(evidence.Repo),
		normalizeSemanticPath(evidence.Path),
		strings.TrimSpace(evidence.Ref),
		lines,
		strings.TrimSpace(evidence.ExcerptHash),
	}, "|")
}

func canonicalSemanticIDSortKey(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if id == "" {
		return "~"
	}
	return id
}

func appendSemanticAlias(aliases []string, candidate string, winnerID string) []string {
	candidate = strings.TrimSpace(candidate)
	winnerID = strings.TrimSpace(winnerID)
	if candidate == "" || candidate == winnerID {
		return aliases
	}
	return dedupeExactStrings(append(aliases, candidate))
}

func dedupeExactStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func stripGeneratedRepoSuffix(value string) string {
	base := strings.TrimSpace(filepath.Base(strings.TrimSpace(value)))
	lastDash := strings.LastIndex(base, "-")
	if lastDash <= 0 || lastDash == len(base)-1 {
		return strings.TrimSpace(value)
	}
	suffix := base[lastDash+1:]
	if len(suffix) < 7 || !isLikelyHexToken(suffix) {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(base[:lastDash])
}

func isLikelyHexToken(value string) bool {
	for _, r := range value {
		isDigit := r >= '0' && r <= '9'
		isLowerHex := r >= 'a' && r <= 'f'
		isUpperHex := r >= 'A' && r <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

func (e *pipelineExecution) effectiveSemanticSnapshot() contracts.SemanticSnapshot {
	base := contracts.SemanticSnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	if e.semanticBase != nil {
		base = *e.semanticBase
	}
	effective := base
	effective.Coverage = base.Coverage
	if e.coverage != nil {
		effective.Coverage = *mergeCoverage(&base.Coverage, e.coverage)
	}
	effective.Questions = mergeQuestions(base.Questions, e.questions)
	effective.Findings = mergeFindings(base.Findings, e.findings)
	return effective
}

func mergeFindings(existing []contracts.Finding, incoming []contracts.Finding) []contracts.Finding {
	merged := map[string]contracts.Finding{}
	for _, finding := range existing {
		if trimmed := strings.TrimSpace(finding.ID); trimmed != "" {
			merged[trimmed] = finding
		}
	}
	for _, finding := range incoming {
		if trimmed := strings.TrimSpace(finding.ID); trimmed != "" {
			merged[trimmed] = finding
		}
	}
	findings := make([]contracts.Finding, 0, len(merged))
	for _, finding := range merged {
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	return findings
}

func aggregateDocumentInfos(manifests []contracts.ShardPackManifest) map[string]*aggregatedDocumentInfo {
	aggregatedDocs := map[string]*aggregatedDocumentInfo{}
	for _, manifest := range manifests {
		for _, document := range manifest.Documents {
			key := strings.TrimSpace(document.CanonicalPath)
			if key == "" {
				continue
			}
			info, ok := aggregatedDocs[key]
			if !ok {
				info = &aggregatedDocumentInfo{
					Kind:              strings.TrimSpace(document.Kind),
					Title:             strings.TrimSpace(document.Title),
					Topics:            map[string]struct{}{},
					CitationIDs:       map[string]struct{}{},
					SourceShards:      map[string]struct{}{},
					SourceDocumentIDs: map[string]struct{}{},
				}
				aggregatedDocs[key] = info
			}
			for _, topic := range document.Topics {
				if trimmed := strings.TrimSpace(topic); trimmed != "" {
					info.Topics[trimmed] = struct{}{}
				}
			}
			for _, citationID := range document.CitationIDs {
				if trimmed := strings.TrimSpace(citationID); trimmed != "" {
					info.CitationIDs[trimmed] = struct{}{}
				}
			}
			if shardID := strings.TrimSpace(manifest.ShardID); shardID != "" {
				info.SourceShards[shardID] = struct{}{}
			}
			if documentID := strings.TrimSpace(document.ID); documentID != "" {
				info.SourceDocumentIDs[documentID] = struct{}{}
			}
			if strings.TrimSpace(info.Title) == "" {
				info.Title = strings.TrimSpace(document.Title)
			}
			if strings.TrimSpace(info.Kind) == "" {
				info.Kind = strings.TrimSpace(document.Kind)
			}
		}
	}
	for canonicalPath, info := range aggregatedDocs {
		info.CanonicalID = preferredCanonicalDocumentID(canonicalPath, setKeysSorted(info.SourceDocumentIDs))
	}
	assignUniqueCanonicalDocumentIDs(aggregatedDocs)
	return aggregatedDocs
}

func preferredCanonicalDocumentID(canonicalPath string, sourceDocumentIDs []string) string {
	for _, documentID := range sourceDocumentIDs {
		if trimmed := strings.TrimSpace(documentID); trimmed != "" {
			return trimmed
		}
	}
	documentID := "doc." + slugutil.Slugify(strings.TrimSpace(canonicalPath))
	if strings.TrimSpace(documentID) == "doc." {
		return "doc.unknown"
	}
	return documentID
}

func assignUniqueCanonicalDocumentIDs(documentInfos map[string]*aggregatedDocumentInfo) {
	sourceIDPaths := map[string]map[string]struct{}{}
	canonicalPaths := make([]string, 0, len(documentInfos))
	for canonicalPath, info := range documentInfos {
		canonicalPath = strings.TrimSpace(canonicalPath)
		if canonicalPath == "" || info == nil {
			continue
		}
		canonicalPaths = append(canonicalPaths, canonicalPath)
		for sourceID := range info.SourceDocumentIDs {
			sourceID = strings.TrimSpace(sourceID)
			if sourceID == "" {
				continue
			}
			if _, ok := sourceIDPaths[sourceID]; !ok {
				sourceIDPaths[sourceID] = map[string]struct{}{}
			}
			sourceIDPaths[sourceID][canonicalPath] = struct{}{}
		}
	}
	sort.Strings(canonicalPaths)

	used := map[string]struct{}{}
	for _, canonicalPath := range canonicalPaths {
		info := documentInfos[canonicalPath]
		if info == nil {
			continue
		}
		base := ""
		for _, sourceID := range setKeysSorted(info.SourceDocumentIDs) {
			if len(sourceIDPaths[sourceID]) == 1 {
				base = sourceID
				break
			}
		}
		if base == "" {
			base = preferredCanonicalDocumentID(canonicalPath, nil)
		}
		info.CanonicalID = uniqueCanonicalDocumentID(base, canonicalPath, used)
	}
}

func uniqueCanonicalDocumentID(base string, canonicalPath string, used map[string]struct{}) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = preferredCanonicalDocumentID(canonicalPath, nil)
	}
	if _, exists := used[base]; !exists {
		used[base] = struct{}{}
		return base
	}
	hash := shortDocumentIDHash(canonicalPath)
	candidate := base + "-" + hash
	if _, exists := used[candidate]; !exists {
		used[candidate] = struct{}{}
		return candidate
	}
	for suffix := 2; ; suffix++ {
		candidate = fmt.Sprintf("%s-%s-%d", base, hash, suffix)
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
	}
}

func shortDocumentIDHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:10]
}

func aggregateCitationIndex(
	runID string,
	generatedAt time.Time,
	manifests []contracts.ShardPackManifest,
	documentInfos map[string]*aggregatedDocumentInfo,
) contracts.CitationIndex {
	merged := map[string]contracts.DocumentCitation{}
	for _, manifest := range manifests {
		remappedDocumentIDs := map[string][]string{}
		for _, document := range manifest.Documents {
			sourceID := strings.TrimSpace(document.ID)
			canonicalPath := strings.TrimSpace(document.CanonicalPath)
			if sourceID == "" || canonicalPath == "" {
				continue
			}
			info := documentInfos[canonicalPath]
			if info == nil || strings.TrimSpace(info.CanonicalID) == "" {
				continue
			}
			remappedDocumentIDs[sourceID] = append(remappedDocumentIDs[sourceID], info.CanonicalID)
		}
		for _, citation := range manifest.Citations {
			citation.DocumentIDs = remapCitationDocumentIDs(citation.DocumentIDs, remappedDocumentIDs)
			existing, ok := merged[citation.ID]
			if !ok {
				existing = citation
			} else {
				existing.ClaimIDs = uniqueSorted(append(existing.ClaimIDs, citation.ClaimIDs...))
				existing.DocumentIDs = uniqueSorted(append(existing.DocumentIDs, citation.DocumentIDs...))
			}
			merged[citation.ID] = existing
		}
	}
	citations := make([]contracts.DocumentCitation, 0, len(merged))
	for _, citation := range merged {
		citations = append(citations, citation)
	}
	sort.Slice(citations, func(i, j int) bool { return citations[i].ID < citations[j].ID })
	return contracts.CitationIndex{
		Version:     1,
		RunID:       runID,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Citations:   append([]contracts.DocumentCitation{}, citations...),
	}
}

func remapCitationDocumentIDs(documentIDs []string, remapped map[string][]string) []string {
	seen := map[string]struct{}{}
	mapped := make([]string, 0, len(documentIDs))
	for _, documentID := range documentIDs {
		documentID = strings.TrimSpace(documentID)
		if documentID == "" {
			continue
		}
		targets := remapped[documentID]
		if len(targets) == 0 {
			targets = []string{documentID}
		}
		for _, target := range targets {
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}
			if _, exists := seen[target]; exists {
				continue
			}
			seen[target] = struct{}{}
			mapped = append(mapped, target)
		}
	}
	sort.Strings(mapped)
	return mapped
}

func buildFinalRunIndex(
	runID string,
	pipeline string,
	generatedAt time.Time,
	stageArtifacts []Artifact,
	manifests []contracts.ShardPackManifest,
	documentInfos map[string]*aggregatedDocumentInfo,
	citationIndex contracts.CitationIndex,
	semantic contracts.SemanticSnapshot,
) (contracts.FinalRunIndex, error) {
	allShardIDs := map[string]struct{}{}
	for _, manifest := range manifests {
		if shardID := strings.TrimSpace(manifest.ShardID); shardID != "" {
			allShardIDs[shardID] = struct{}{}
		}
	}
	allCitationIDs := make([]string, 0, len(citationIndex.Citations))
	for _, citation := range citationIndex.Citations {
		if trimmed := strings.TrimSpace(citation.ID); trimmed != "" {
			allCitationIDs = append(allCitationIDs, trimmed)
		}
	}
	sort.Strings(allCitationIDs)
	allSourceShards := setKeysSorted(allShardIDs)

	documents := make([]contracts.FinalRunDocument, 0, len(stageArtifacts))
	topicDocs := map[string]map[string]struct{}{}
	for _, artifact := range stageArtifacts {
		if strings.Contains(artifact.Path, "/staging/") == false {
			continue
		}
		canonicalPath := stripStagePrefix(artifact.Path)
		info := documentInfos[canonicalPath]
		documentID := preferredCanonicalDocumentID(canonicalPath, nil)
		entry := contracts.FinalRunDocument{
			ID:            documentID,
			Kind:          artifact.Kind,
			Title:         artifact.Label,
			CanonicalPath: canonicalPath,
			StagedPath:    artifact.Path,
			Topics:        []string{},
			CitationIDs:   []string{},
			SourceShards:  []string{},
			Status:        "staged",
		}
		if info != nil {
			if strings.TrimSpace(info.CanonicalID) != "" {
				entry.ID = strings.TrimSpace(info.CanonicalID)
			}
			if strings.TrimSpace(info.Kind) != "" {
				entry.Kind = info.Kind
			}
			if strings.TrimSpace(info.Title) != "" {
				entry.Title = info.Title
			}
			entry.Topics = setKeysSorted(info.Topics)
			entry.CitationIDs = setKeysSorted(info.CitationIDs)
			entry.SourceShards = setKeysSorted(info.SourceShards)
		}
		if len(entry.CitationIDs) == 0 && requiresDocumentCitations(entry) {
			entry.CitationIDs = append([]string{}, allCitationIDs...)
			entry.Topics = appendUniqueStrings(entry.Topics, "runtime-derived")
		}
		if len(entry.SourceShards) == 0 {
			if len(allSourceShards) > 0 {
				entry.SourceShards = append([]string{}, allSourceShards...)
			} else {
				entry.SourceShards = []string{"aggregated"}
			}
		}
		for _, topic := range entry.Topics {
			if _, ok := topicDocs[topic]; !ok {
				topicDocs[topic] = map[string]struct{}{}
			}
			topicDocs[topic][entry.ID] = struct{}{}
		}
		documents = append(documents, entry)
	}
	sort.Slice(documents, func(i, j int) bool { return documents[i].CanonicalPath < documents[j].CanonicalPath })

	topics := make([]contracts.TopicIndexEntry, 0, len(topicDocs))
	for topicID, documentIDs := range topicDocs {
		topics = append(topics, contracts.TopicIndexEntry{
			ID:          topicID,
			DocumentIDs: setKeysSorted(documentIDs),
		})
	}
	sort.Slice(topics, func(i, j int) bool { return topics[i].ID < topics[j].ID })

	index := contracts.FinalRunIndex{
		Version:            1,
		RunID:              runID,
		Pipeline:           pipeline,
		GeneratedAt:        generatedAt.UTC().Format(time.RFC3339),
		CitationIndexPath:  runtimeCitationIndexPath(runID),
		CanonicalDocuments: append([]contracts.FinalRunDocument{}, documents...),
		Topics:             append([]contracts.TopicIndexEntry{}, topics...),
		Semantic:           semantic,
	}
	raw, err := json.Marshal(index)
	if err != nil {
		return contracts.FinalRunIndex{}, err
	}
	parsed, err := contracts.ParseFinalRunIndex(raw)
	if err != nil {
		return contracts.FinalRunIndex{}, err
	}
	return parsed, nil
}

func reconcileRuntimeDerivedCitationDocuments(citationIndex contracts.CitationIndex, finalIndex contracts.FinalRunIndex) (contracts.CitationIndex, bool) {
	citationOffsets := map[string]int{}
	for idx, citation := range citationIndex.Citations {
		citationID := strings.TrimSpace(citation.ID)
		if citationID == "" {
			continue
		}
		citationOffsets[citationID] = idx
	}

	changed := false
	for _, document := range finalIndex.CanonicalDocuments {
		if !containsTrimmedString(document.Topics, "runtime-derived") {
			continue
		}
		documentID := strings.TrimSpace(document.ID)
		if documentID == "" {
			continue
		}
		for _, rawCitationID := range document.CitationIDs {
			citationID := strings.TrimSpace(rawCitationID)
			if citationID == "" {
				continue
			}
			offset, ok := citationOffsets[citationID]
			if !ok {
				continue
			}
			if containsTrimmedString(citationIndex.Citations[offset].DocumentIDs, documentID) {
				continue
			}
			citationIndex.Citations[offset].DocumentIDs = uniqueSorted(append(citationIndex.Citations[offset].DocumentIDs, documentID))
			changed = true
		}
	}
	return citationIndex, changed
}

func stripStagePrefix(stagedPath string) string {
	parts := strings.Split(filepath.ToSlash(strings.TrimSpace(stagedPath)), "/staging/final/")
	if len(parts) == 2 {
		return parts[1]
	}
	return filepath.ToSlash(strings.TrimSpace(stagedPath))
}

func setKeysSorted(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(values))
	for value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			keys = append(keys, trimmed)
		}
	}
	sort.Strings(keys)
	return keys
}

func (e *pipelineExecution) validateStagedArtifacts() []contracts.ValidatorIssue {
	issues := []contracts.ValidatorIssue{}
	if e.finalRunIndex == nil {
		return []contracts.ValidatorIssue{{
			Code:     "final_run_index_missing",
			Severity: "error",
			Message:  "final run index was not assembled",
		}}
	}
	if e.citationIndex == nil {
		return []contracts.ValidatorIssue{{
			Code:     "citation_index_missing",
			Severity: "error",
			Message:  "citation index was not assembled",
		}}
	}
	expectedRunID := strings.TrimSpace(e.runID)
	if expectedRunID == "" {
		expectedRunID = strings.TrimSpace(e.finalRunIndex.RunID)
	}
	if strings.TrimSpace(e.finalRunIndex.RunID) != expectedRunID ||
		strings.TrimSpace(e.citationIndex.RunID) != expectedRunID {
		issues = append(issues, contracts.ValidatorIssue{
			Code:     "foreign_run_index",
			Severity: "error",
			Message:  fmt.Sprintf("final/citation index run ids must both equal %q", expectedRunID),
		})
	}
	expectedCitationPath := runtimeCitationIndexPath(expectedRunID)
	if strings.TrimSpace(e.finalRunIndex.CitationIndexPath) != expectedCitationPath {
		issues = append(issues, contracts.ValidatorIssue{
			Code:     "foreign_run_citation_index",
			Severity: "error",
			Message:  fmt.Sprintf("citation index path must be %q", expectedCitationPath),
			Path:     e.finalRunIndex.CitationIndexPath,
		})
	}
	if err := artifactquality.ValidateSemanticEnvelope(e.finalRunIndex.Semantic); err != nil {
		issues = append(issues, contracts.ValidatorIssue{Code: "semantic_envelope_invalid", Severity: "error", Message: err.Error()})
	}
	shardSemantics := make([]contracts.SemanticSnapshot, 0, len(e.shardPacks))
	for _, manifest := range e.shardPacks {
		shardSemantics = append(shardSemantics, manifest.Semantic)
	}
	if err := artifactquality.ValidateSemanticIDCollisions(shardSemantics...); err != nil {
		issues = append(issues, contracts.ValidatorIssue{Code: "semantic_id_collision", Severity: "error", Message: err.Error()})
	}

	citationIDs := map[string]struct{}{}
	citationsByID := map[string]contracts.DocumentCitation{}
	claimIDs := map[string]struct{}{}
	strictCitationChecks := len(e.shardPacks) > 0
	for _, citation := range e.citationIndex.Citations {
		citationID := strings.TrimSpace(citation.ID)
		if _, exists := citationIDs[citation.ID]; exists {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "duplicate_citation_id",
				Severity:   "error",
				Message:    fmt.Sprintf("duplicate citation id %q", citation.ID),
				CitationID: citation.ID,
			})
		}
		citationIDs[citation.ID] = struct{}{}
		if citationID != "" {
			citationsByID[citationID] = citation
		}
		for _, claimID := range citation.ClaimIDs {
			if _, exists := claimIDs[claimID]; exists {
				issues = append(issues, contracts.ValidatorIssue{
					Code:       "duplicate_claim_id",
					Severity:   "error",
					Message:    fmt.Sprintf("duplicate claim id %q", claimID),
					CitationID: citation.ID,
				})
			}
			claimIDs[claimID] = struct{}{}
		}
		if len(e.resolvedRepoPaths) > 0 {
			repoRoot := resolvedCitationRepoRoot(e.resolvedRepoPaths, citation.Repo)
			switch {
			case repoRoot == "":
				issues = append(issues, contracts.ValidatorIssue{
					Code:       "citation_repo_unknown",
					Severity:   "error",
					Message:    fmt.Sprintf("citation %q references unknown repository %q", citation.ID, citation.Repo),
					CitationID: citation.ID,
				})
			case validateCitationEvidenceFile(repoRoot, citation.Path) != nil:
				issues = append(issues, contracts.ValidatorIssue{
					Code:       "citation_evidence_unavailable",
					Severity:   "error",
					Message:    fmt.Sprintf("citation %q evidence path %q is not a concrete in-root file", citation.ID, citation.Path),
					Path:       citation.Path,
					CitationID: citation.ID,
				})
			case citationHasBoundedEvidence(citation):
				if err := validateCitationEvidenceContent(repoRoot, citation); err != nil {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       evidence.Code(err),
						Severity:   "error",
						Message:    fmt.Sprintf("citation %q evidence bytes failed bounded validation: %v", citation.ID, err),
						Path:       citation.Path,
						CitationID: citation.ID,
					})
				}
			}
		}
	}
	if len(e.resolvedRepoPaths) > 0 {
		repoAliases := newSemanticRepoAliasResolver(e.resolvedRepoPaths, e.shardPacks)
		evidencePaths := newSemanticEvidencePathResolver(e.resolvedRepoPaths, repoAliases)
		for _, entity := range e.finalRunIndex.Semantic.Entities {
			for _, source := range entity.Provenance.Evidence {
				if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
					appendSemanticEvidenceIssue(&issues, evidencePaths, repoAliases, source)
				}
			}
		}
		for _, edge := range e.finalRunIndex.Semantic.Edges {
			for _, source := range edge.Provenance.Evidence {
				if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
					appendSemanticEvidenceIssue(&issues, evidencePaths, repoAliases, source)
				}
			}
		}
		for _, finding := range e.finalRunIndex.Semantic.Findings {
			for _, source := range finding.Provenance.Evidence {
				if evidence.HasBoundedEvidence(source.Lines, source.Excerpt, source.ExcerptHash) {
					appendSemanticEvidenceIssue(&issues, evidencePaths, repoAliases, source)
				}
			}
		}
	}

	seenTopics := map[string]struct{}{}
	documentsByID := map[string]contracts.FinalRunDocument{}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		documentsByID[document.ID] = document
		expectedStageRoot := runtimeFinalArtifactRoot(expectedRunID)
		stagedPath := path.Clean(filepath.ToSlash(strings.TrimSpace(document.StagedPath)))
		if stagedPath == "." || (stagedPath != expectedStageRoot && !strings.HasPrefix(stagedPath, expectedStageRoot+"/")) {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "foreign_run_staged_document",
				Severity:   "error",
				Message:    fmt.Sprintf("document %q staged path is outside run %q", document.ID, expectedRunID),
				Path:       document.StagedPath,
				DocumentID: document.ID,
			})
		}
		if strictCitationChecks && requiresDocumentCitations(document) && len(document.CitationIDs) == 0 {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "missing_document_citations",
				Severity:   "error",
				Message:    fmt.Sprintf("document %q has no citation ids", document.CanonicalPath),
				Path:       document.CanonicalPath,
				DocumentID: document.ID,
			})
		}
		for _, citationID := range document.CitationIDs {
			if strictCitationChecks {
				citation, ok := citationsByID[strings.TrimSpace(citationID)]
				if !ok {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       "unknown_document_citation",
						Severity:   "error",
						Message:    fmt.Sprintf("document %q references unknown citation %q", document.CanonicalPath, citationID),
						Path:       document.CanonicalPath,
						DocumentID: document.ID,
						CitationID: citationID,
					})
				} else if !containsTrimmedString(citation.DocumentIDs, document.ID) {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       "asymmetric_document_citation",
						Severity:   "error",
						Message:    fmt.Sprintf("document %q references citation %q but citation does not list document id %q", document.CanonicalPath, citationID, document.ID),
						Path:       document.CanonicalPath,
						DocumentID: document.ID,
						CitationID: citationID,
					})
				}
			}
		}
		if _, err := e.workspace.ReadFile(document.StagedPath); err != nil {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "missing_staged_document",
				Severity:   "error",
				Message:    fmt.Sprintf("staged document %q is missing", document.StagedPath),
				Path:       document.StagedPath,
				DocumentID: document.ID,
			})
		}
		if len(document.CitationIDs) == 0 && keyDocumentClaimsCitationCompleteness(document, e.workspace) {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "empty_claimed_citation_coverage",
				Severity:   "error",
				Message:    fmt.Sprintf("key document %q claims citation completeness but has no citation coverage", document.CanonicalPath),
				Path:       document.CanonicalPath,
				DocumentID: document.ID,
			})
		}
	}
	for _, topic := range e.finalRunIndex.Topics {
		if _, exists := seenTopics[topic.ID]; exists {
			issues = append(issues, contracts.ValidatorIssue{
				Code:     "duplicate_topic_id",
				Severity: "error",
				Message:  fmt.Sprintf("duplicate topic id %q", topic.ID),
			})
		}
		seenTopics[topic.ID] = struct{}{}
		for _, documentID := range topic.DocumentIDs {
			if _, ok := documentsByID[documentID]; !ok {
				issues = append(issues, contracts.ValidatorIssue{
					Code:     "broken_topic_reference",
					Severity: "error",
					Message:  fmt.Sprintf("topic %q references unknown document %q", topic.ID, documentID),
				})
			}
		}
	}
	if strictCitationChecks {
		for _, citation := range e.citationIndex.Citations {
			citationID := strings.TrimSpace(citation.ID)
			if citationID == "" {
				continue
			}
			for _, rawDocumentID := range citation.DocumentIDs {
				documentID := strings.TrimSpace(rawDocumentID)
				if documentID == "" {
					continue
				}
				document, ok := documentsByID[documentID]
				if !ok {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       "unknown_citation_document",
						Severity:   "error",
						Message:    fmt.Sprintf("citation %q references unknown document %q", citationID, documentID),
						DocumentID: documentID,
						CitationID: citationID,
					})
					continue
				}
				if !containsTrimmedString(document.CitationIDs, citationID) {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       "asymmetric_citation_document",
						Severity:   "error",
						Message:    fmt.Sprintf("citation %q references document %q but document does not list citation id %q", citationID, documentID, citationID),
						Path:       document.CanonicalPath,
						DocumentID: documentID,
						CitationID: citationID,
					})
				}
			}
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

func resolvedCitationRepoRoot(repoPaths map[string]string, repo string) string {
	target := strings.ToLower(strings.TrimSpace(repo))
	for name, root := range repoPaths {
		if strings.ToLower(strings.TrimSpace(name)) == target {
			return strings.TrimSpace(root)
		}
	}
	return ""
}

func validateCitationEvidenceFile(repoRoot string, rawPath string) error {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath)))
	if relative == "." || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("invalid relative evidence path")
	}
	resolvedRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		return err
	}
	resolvedTarget, err := filepath.EvalSymlinks(filepath.Join(repoRoot, relative))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("evidence path escapes repository")
	}
	info, err := os.Stat(resolvedTarget)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("evidence path is not a regular file")
	}
	return nil
}

func citationHasBoundedEvidence(citation contracts.DocumentCitation) bool {
	return evidence.HasBoundedEvidence(citation.Lines, citation.Excerpt, citation.ExcerptHash)
}

func validateCitationEvidenceContent(repoRoot string, citation contracts.DocumentCitation) error {
	return validateEvidenceContent(repoRoot, citation.Path, citation.Lines, citation.Excerpt, citation.ExcerptHash)
}

func validateEvidenceContent(repoRoot, rawPath string, lines *contracts.LineRange, excerpt, excerptHash string) error {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	relative := filepath.Clean(filepath.FromSlash(strings.TrimSpace(rawPath)))
	resolvedTarget, err := filepath.EvalSymlinks(filepath.Join(repoRoot, relative))
	if err != nil {
		return err
	}
	return evidence.ValidateFile(resolvedTarget, lines, excerpt, excerptHash)
}

func appendSemanticEvidenceIssue(issues *[]contracts.ValidatorIssue, evidencePaths semanticEvidencePathResolver, repoAliases semanticRepoAliasResolver, source contracts.Evidence) {
	canonicalRepo := repoAliases.canonical(source.Repo)
	repoRoot := strings.TrimSpace(evidencePaths.rootsByRepo[strings.ToLower(canonicalRepo)])
	if repoRoot == "" {
		*issues = append(*issues, contracts.ValidatorIssue{
			Code: "evidence_repo_unknown", Severity: "error", Message: fmt.Sprintf("semantic evidence references unknown repository %q", source.Repo), Path: source.Path,
		})
		return
	}
	resolvedPath := evidencePaths.resolve(canonicalRepo, source.Path)
	if err := validateCitationEvidenceFile(repoRoot, resolvedPath); err != nil {
		*issues = append(*issues, contracts.ValidatorIssue{
			Code: "evidence_unavailable", Severity: "error", Message: fmt.Sprintf("semantic evidence path %q is not a concrete in-root file", source.Path), Path: source.Path,
		})
		return
	}
	if citationHasBoundedEvidence(contracts.DocumentCitation{Lines: source.Lines, Excerpt: source.Excerpt, ExcerptHash: source.ExcerptHash}) {
		if err := validateEvidenceContent(repoRoot, resolvedPath, source.Lines, source.Excerpt, source.ExcerptHash); err != nil {
			*issues = append(*issues, contracts.ValidatorIssue{
				Code: evidence.Code(err), Severity: "error", Message: fmt.Sprintf("semantic evidence bytes failed bounded validation: %v", err), Path: source.Path,
			})
		}
	}
}

func keyDocumentClaimsCitationCompleteness(document contracts.FinalRunDocument, ws workspace.Root) bool {
	canonical := filepath.ToSlash(strings.TrimSpace(document.CanonicalPath))
	isKey := canonical == "reports/as-is/overview.md" ||
		strings.HasPrefix(canonical, "reports/findings/") ||
		strings.HasPrefix(canonical, "proposals/")
	if !isKey {
		return false
	}
	raw, err := ws.ReadFile(document.StagedPath)
	if err != nil {
		return false
	}
	normalized := strings.ToLower(string(raw))
	for _, phrase := range []string{
		"citation coverage is complete",
		"citations are complete",
		"fully cited",
		"all claims are cited",
		"complete evidence coverage",
	} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}

func containsTrimmedString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func requiresDocumentCitations(document contracts.FinalRunDocument) bool {
	switch document.Kind {
	case "diagram", "diagram-index", "taskrun":
		return false
	}
	if strings.HasSuffix(document.CanonicalPath, ".task-envelope.json") {
		return false
	}
	return true
}
