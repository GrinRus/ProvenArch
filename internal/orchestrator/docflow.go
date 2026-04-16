package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	"github.com/GrinRus/ProvenArch/internal/reports"
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
	Kind         string
	Title        string
	Topics       map[string]struct{}
	CitationIDs  map[string]struct{}
	SourceShards map[string]struct{}
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
		return "validator"
	default:
		return "runtime"
	}
}

func (e *pipelineExecution) runtimeArtifactContext(stepID string, shardID string, repoScopes []string) (string, string, []string, error) {
	var rel string
	switch {
	case strings.HasSuffix(stepID, "step1.collect"):
		rel = runtimeShardArtifactRoot(e.runID, shardID)
	case strings.HasSuffix(stepID, "step3.findings"):
		rel = runtimeValidatorArtifactRoot(e.runID)
	default:
		rel = path.Join("reports", "taskruns", e.runID, "runtime")
	}
	abs, err := e.workspace.Resolve(rel)
	if err != nil {
		return "", "", nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", "", nil, fmt.Errorf("create runtime artifact root: %w", err)
	}

	roots := []string{e.workspace.Path}
	if strings.HasSuffix(stepID, "step3.findings") {
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
	return rel, abs, normalizeOrderedUniqueStrings(roots), nil
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

func (e *pipelineExecution) assembleStagedDocFlow() error {
	stageRootRel := runtimeFinalArtifactRoot(e.runID)
	stageRootAbs, err := e.workspace.Resolve(stageRootRel)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(stageRootAbs); err != nil {
		return fmt.Errorf("reset staged final root: %w", err)
	}
	if err := os.MkdirAll(stageRootAbs, 0o755); err != nil {
		return fmt.Errorf("create staged final root: %w", err)
	}
	stageRoot := workspace.Root{Path: stageRootAbs}

	baseCompatibility := aggregateCompatibilitySnapshot(e.shardPacks)
	e.compatibilityBase = &baseCompatibility
	compatibility := e.effectiveCompatibilitySnapshot()
	stageStore := model.NewStore(stageRoot)
	if _, err := stageStore.ApplyChangeset(buildCompatibilityModelTaskResult(e.runID, compatibility)); err != nil {
		return fmt.Errorf("apply staged compatibility model: %w", err)
	}
	entities, err := stageStore.ListEntities()
	if err != nil {
		return err
	}
	edges, err := stageStore.ListEdges()
	if err != nil {
		return err
	}

	stageCompiler := reports.NewCompiler(stageRoot)
	renderCtx := e.renderContext()
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

	authoredDocs, err := collectAuthoredStageDocuments(e.shardPacks)
	if err != nil {
		return err
	}
	for _, document := range authoredDocs {
		content := document.Content
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := stageRoot.WriteFile(document.CanonicalPath, []byte(content)); err != nil {
			return err
		}
		registerStagedArtifact(Artifact{
			Path:  path.Join(stageRootRel, document.CanonicalPath),
			Kind:  document.Kind,
			Label: document.Title,
		})
	}

	hasCanonicalPrefix := func(prefix string) bool {
		for canonicalPath := range canonicalPaths {
			if strings.HasPrefix(canonicalPath, prefix) {
				return true
			}
		}
		return false
	}
	hasCanonicalPath := func(target string) bool {
		_, ok := canonicalPaths[target]
		return ok
	}
	hasCanonicalSuffix := func(suffix string) bool {
		for canonicalPath := range canonicalPaths {
			if strings.HasSuffix(canonicalPath, suffix) {
				return true
			}
		}
		return false
	}

	// Narratives are docs-first from runtime-authored staged packs.
	// Compiler paths remain compatibility fallback only when required surfaces are absent.
	if !hasCanonicalPrefix("reports/as-is/") {
		if err := registerCompiledArtifacts(stageCompiler.CompileAsIs(entities, edges, renderCtx)); err != nil {
			return err
		}
	}
	if !hasCanonicalPrefix("reports/coverage/") {
		if err := registerCompiledArtifacts(stageCompiler.WriteCoverage(&compatibility.Coverage, compatibility.Questions, renderCtx)); err != nil {
			return err
		}
	}
	if !hasCanonicalPrefix("reports/findings/") {
		if err := registerCompiledArtifacts(stageCompiler.WriteFindings(compatibility.Findings, renderCtx)); err != nil {
			return err
		}
	}
	if !hasCanonicalPrefix("proposals/") {
		if err := registerCompiledArtifacts(stageCompiler.CompileProposals(compatibility.Findings, renderCtx)); err != nil {
			return err
		}
	}
	if !hasCanonicalPath("reports/agent-outputs/architect/summary.md") {
		if err := registerCompiledArtifacts(stageCompiler.WriteArchitectSummary(e.renderArchitectSummary(), renderCtx)); err != nil {
			return err
		}
	}
	if !hasCanonicalPrefix("reports/agent-outputs/domains/") {
		if domainReports, domainErr := e.authoredDomainReports(); domainErr != nil {
			return domainErr
		} else if err := registerCompiledArtifacts(stageCompiler.WriteDomainOutputs(domainReports)); err != nil {
			return err
		}
	}
	if !hasCanonicalSuffix(".task-envelope.json") {
		if err := registerCompiledArtifacts(stageCompiler.WriteDomainTaskEnvelopes(e.stagedDomainEnvelopes())); err != nil {
			return err
		}
	}

	if err := registerCompiledArtifacts(stageCompiler.CompileC4Diagrams(entities, edges)); err != nil {
		return err
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

	citationIndex := aggregateCitationIndex(e.runID, e.clock().UTC(), e.shardPacks)
	citationRaw, err := json.MarshalIndent(citationIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal citation index: %w", err)
	}
	citationRaw = append(citationRaw, '\n')
	if err := stageRoot.WriteFile(citationIndexFile, citationRaw); err != nil {
		return err
	}
	parsedCitationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return err
	}

	for _, artifact := range stageArtifacts {
		e.addArtifacts(artifact)
	}
	e.addArtifacts(Artifact{
		Path:  runtimeCitationIndexPath(e.runID),
		Kind:  "taskrun",
		Label: "Citation Index",
	})

	finalRunIndex, err := buildFinalRunIndex(
		e.runID,
		string(e.pipeline),
		e.clock().UTC(),
		stageArtifacts,
		e.shardPacks,
		parsedCitationIndex,
		compatibility,
	)
	if err != nil {
		return err
	}
	finalIndexRaw, err := json.MarshalIndent(finalRunIndex, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal final run index: %w", err)
	}
	finalIndexRaw = append(finalIndexRaw, '\n')
	if err := stageRoot.WriteFile(finalRunIndexFile, finalIndexRaw); err != nil {
		return err
	}
	parsedFinalRunIndex, err := contracts.ParseFinalRunIndex(finalIndexRaw)
	if err != nil {
		return err
	}

	e.addArtifacts(Artifact{
		Path:  runtimeFinalRunIndexPath(e.runID),
		Kind:  "taskrun",
		Label: "Final Run Index",
	})
	e.finalRunIndex = &parsedFinalRunIndex
	e.citationIndex = &parsedCitationIndex
	e.findings = append([]contracts.Finding(nil), compatibility.Findings...)
	e.questions = append([]contracts.Question(nil), compatibility.Questions...)
	e.coverage = mergeCoverage(nil, &compatibility.Coverage)
	e.logInfo(e.stepStatus.CurrentStep, "", "staged doc flow assembled", map[string]any{
		"shard_packs":    len(e.shardPacks),
		"staged_docs":    len(parsedFinalRunIndex.CanonicalDocuments),
		"citation_count": len(parsedCitationIndex.Citations),
		"entities":       len(entities),
		"edges":          len(edges),
		"findings":       len(compatibility.Findings),
		"questions":      len(compatibility.Questions),
	})
	return nil
}

func (e *pipelineExecution) authoredDomainReports() (map[string]string, error) {
	reportsByDomain := map[string]string{}
	for _, manifest := range e.shardPacks {
		for _, document := range manifest.Documents {
			if !strings.HasPrefix(strings.TrimSpace(document.CanonicalPath), "reports/agent-outputs/domains/") {
				continue
			}
			content, err := readShardDocument(manifest, document)
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

func readShardDocument(manifest contracts.ShardPackManifest, document contracts.AuthoredDocument) (string, error) {
	artifactRoot := strings.TrimSpace(manifest.ArtifactRoot)
	if artifactRoot == "" {
		return "", fmt.Errorf("read shard document %q: manifest artifact_root is empty", document.ID)
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

func collectAuthoredStageDocuments(manifests []contracts.ShardPackManifest) ([]stagedAuthoredDocument, error) {
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
			content, err := readShardDocument(manifest, document)
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

func aggregateCompatibilitySnapshot(manifests []contracts.ShardPackManifest) contracts.CompatibilitySnapshot {
	snapshot := contracts.CompatibilitySnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	entityByID := map[string]contracts.Entity{}
	edgeByID := map[string]contracts.Edge{}
	findingByID := map[string]contracts.Finding{}

	for _, manifest := range manifests {
		snapshot.Coverage = *mergeCoverage(&snapshot.Coverage, &manifest.Compatibility.Coverage)
		snapshot.Questions = mergeQuestions(snapshot.Questions, manifest.Compatibility.Questions)
		for _, entity := range manifest.Compatibility.Entities {
			entityByID[entity.ID] = entity
		}
		for _, edge := range manifest.Compatibility.Edges {
			edgeByID[edge.ID] = edge
		}
		for _, finding := range manifest.Compatibility.Findings {
			findingByID[finding.ID] = finding
		}
	}

	for _, entity := range entityByID {
		snapshot.Entities = append(snapshot.Entities, entity)
	}
	for _, edge := range edgeByID {
		snapshot.Edges = append(snapshot.Edges, edge)
	}
	for _, finding := range findingByID {
		snapshot.Findings = append(snapshot.Findings, finding)
	}
	sort.Slice(snapshot.Entities, func(i, j int) bool { return snapshot.Entities[i].ID < snapshot.Entities[j].ID })
	sort.Slice(snapshot.Edges, func(i, j int) bool { return snapshot.Edges[i].ID < snapshot.Edges[j].ID })
	sort.Slice(snapshot.Findings, func(i, j int) bool { return snapshot.Findings[i].ID < snapshot.Findings[j].ID })
	return snapshot
}

func (e *pipelineExecution) effectiveCompatibilitySnapshot() contracts.CompatibilitySnapshot {
	base := contracts.CompatibilitySnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	if e.compatibilityBase != nil {
		base = *e.compatibilityBase
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

func buildCompatibilityModelTaskResult(runID string, snapshot contracts.CompatibilitySnapshot) contracts.TaskResult {
	changeset := make([]contracts.Operation, 0, len(snapshot.Entities)+len(snapshot.Edges))
	for _, entity := range snapshot.Entities {
		entityCopy := entity
		changeset = append(changeset, contracts.Operation{
			Op:     "upsert_entity",
			Entity: &entityCopy,
		})
	}
	for _, edge := range snapshot.Edges {
		edgeCopy := edge
		changeset = append(changeset, contracts.Operation{
			Op:   "upsert_edge",
			Edge: &edgeCopy,
		})
	}
	return contracts.TaskResult{
		Meta: contracts.Meta{
			TaskID:    "compat-" + runID,
			StepID:    "compat.extract",
			Runtime:   contracts.RuntimeMeta{Name: "docflow-compat", Version: "1"},
			StartedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		},
		Summary:   "Derived compatibility model from final run index.",
		Changeset: changeset,
	}
}

func aggregateCitationIndex(runID string, generatedAt time.Time, manifests []contracts.ShardPackManifest) contracts.CitationIndex {
	merged := map[string]contracts.DocumentCitation{}
	for _, manifest := range manifests {
		for _, citation := range manifest.Citations {
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

func buildFinalRunIndex(
	runID string,
	pipeline string,
	generatedAt time.Time,
	stageArtifacts []Artifact,
	manifests []contracts.ShardPackManifest,
	citationIndex contracts.CitationIndex,
	compatibility contracts.CompatibilitySnapshot,
) (contracts.FinalRunIndex, error) {
	aggregatedDocs := map[string]*aggregatedDocumentInfo{}
	allShardIDs := map[string]struct{}{}
	for _, manifest := range manifests {
		if shardID := strings.TrimSpace(manifest.ShardID); shardID != "" {
			allShardIDs[shardID] = struct{}{}
		}
		for _, document := range manifest.Documents {
			key := strings.TrimSpace(document.CanonicalPath)
			if key == "" {
				continue
			}
			info, ok := aggregatedDocs[key]
			if !ok {
				info = &aggregatedDocumentInfo{
					Kind:         document.Kind,
					Title:        document.Title,
					Topics:       map[string]struct{}{},
					CitationIDs:  map[string]struct{}{},
					SourceShards: map[string]struct{}{},
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
			if strings.TrimSpace(info.Title) == "" {
				info.Title = document.Title
			}
			if strings.TrimSpace(info.Kind) == "" {
				info.Kind = document.Kind
			}
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
		info := aggregatedDocs[canonicalPath]
		documentID := "doc." + slugutil.Slugify(canonicalPath)
		if strings.TrimSpace(documentID) == "doc." {
			documentID = "doc.unknown"
		}
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
		Compatibility:      compatibility,
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

	citationIDs := map[string]struct{}{}
	claimIDs := map[string]struct{}{}
	strictCitationChecks := len(e.shardPacks) > 0
	for _, citation := range e.citationIndex.Citations {
		if _, exists := citationIDs[citation.ID]; exists {
			issues = append(issues, contracts.ValidatorIssue{
				Code:       "duplicate_citation_id",
				Severity:   "error",
				Message:    fmt.Sprintf("duplicate citation id %q", citation.ID),
				CitationID: citation.ID,
			})
		}
		citationIDs[citation.ID] = struct{}{}
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
	}

	seenTopics := map[string]struct{}{}
	documentsByID := map[string]contracts.FinalRunDocument{}
	for _, document := range e.finalRunIndex.CanonicalDocuments {
		documentsByID[document.ID] = document
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
				if _, ok := citationIDs[citationID]; !ok {
					issues = append(issues, contracts.ValidatorIssue{
						Code:       "unknown_document_citation",
						Severity:   "error",
						Message:    fmt.Sprintf("document %q references unknown citation %q", document.CanonicalPath, citationID),
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
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Code == issues[j].Code {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
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

func (e *pipelineExecution) promoteValidatedArtifacts() error {
	if e.finalRunIndex == nil {
		return fmt.Errorf("promote validated artifacts: final run index is missing")
	}
	if e.validatorVerdict == nil {
		return fmt.Errorf("promote validated artifacts: validator verdict is missing")
	}
	if e.validatorVerdict.Verdict != "PASS" {
		return fmt.Errorf("promote validated artifacts: validator verdict is %s", e.validatorVerdict.Verdict)
	}

	for _, document := range e.finalRunIndex.CanonicalDocuments {
		content, err := e.workspace.ReadFile(document.StagedPath)
		if err != nil {
			return fmt.Errorf("read staged artifact %q: %w", document.StagedPath, err)
		}
		if err := e.workspace.WriteFile(document.CanonicalPath, content); err != nil {
			return fmt.Errorf("promote artifact %q: %w", document.CanonicalPath, err)
		}
		e.addArtifacts(Artifact{
			Path:  document.CanonicalPath,
			Kind:  document.Kind,
			Label: document.Title,
		})
	}

	if err := e.rebuildCompatibilityModel(); err != nil {
		return err
	}

	entities, err := e.store.ListEntities()
	if err != nil {
		return err
	}
	edges, err := e.store.ListEdges()
	if err != nil {
		return err
	}
	diagramArtifacts, err := e.compiler.CompileC4Diagrams(entities, edges)
	if err != nil {
		return err
	}
	e.addArtifacts(toOrchestratorArtifacts(diagramArtifacts)...)
	e.logInfo(e.stepStatus.CurrentStep, "", "validated artifacts promoted", map[string]any{
		"canonical_docs": len(e.finalRunIndex.CanonicalDocuments),
	})
	return nil
}

func (e *pipelineExecution) rebuildCompatibilityModel() error {
	if e.finalRunIndex == nil {
		return fmt.Errorf("rebuild compatibility model: final run index is missing")
	}
	for _, rel := range []string{"model/entities", "model/edges"} {
		abs, err := e.workspace.Resolve(rel)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("clear compatibility model dir %q: %w", rel, err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("recreate compatibility model dir %q: %w", rel, err)
		}
	}
	_, err := e.store.ApplyChangeset(buildCompatibilityModelTaskResult(e.runID, e.finalRunIndex.Compatibility))
	return err
}
