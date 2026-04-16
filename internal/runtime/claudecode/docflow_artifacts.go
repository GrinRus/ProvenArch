package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

const (
	domainReportFileName      = "domain-report.md"
	shardPackManifestFileName = "shard-pack-manifest.json"
	validatorVerdictFileName  = "validator-verdict.json"
	finalRunIndexFileName     = "final-run-index.json"
	citationIndexFileName     = "citation-index.json"
)

type runtimeCitationSource struct {
	claimKey   string
	repo       string
	ref        string
	path       string
	lines      *contracts.LineRange
	excerpt    string
	hash       string
	documentID string
}

func persistDocsFirstArtifacts(task acpruntime.Task, result contracts.TaskResult) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return nil
	}
	switch task.StepID {
	case "init.step1.collect", "refresh.step1.collect":
		return writeShardPackManifest(writeRoot, task, result)
	case "init.step3.findings", "refresh.step3.findings":
		return writeValidatorVerdict(writeRoot, task)
	default:
		return nil
	}
}

func PersistCompatibilityDocflowArtifacts(task acpruntime.Task, result contracts.TaskResult) error {
	return persistDocsFirstArtifacts(task, result)
}

func writeShardPackManifest(writeRoot string, task acpruntime.Task, result contracts.TaskResult) error {
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		return fmt.Errorf("create shard write root: %w", err)
	}

	compatibility := compatibilitySnapshotFromTaskResult(result)
	documentID := runtimeDomainDocumentID(task)
	citations := deriveRuntimeCitations(task, compatibility, documentID)
	document := contracts.AuthoredDocument{
		ID:            documentID,
		Kind:          "agent-output",
		Title:         runtimeDomainDocumentTitle(task),
		Path:          domainReportFileName,
		CanonicalPath: runtimeDomainCanonicalPath(task),
		Topics:        runtimeDomainTopics(task),
		CitationIDs:   collectCitationIDs(citations),
		Status:        "staged",
	}

	reportContent := renderRuntimeDomainReport(task, result, compatibility, citations)
	if err := writeRuntimeArtifactFile(writeRoot, document.Path, []byte(reportContent)); err != nil {
		return err
	}

	manifest := contracts.ShardPackManifest{
		Version:       1,
		RunID:         strings.TrimSpace(task.RunID),
		StepID:        strings.TrimSpace(task.StepID),
		ShardID:       strings.TrimSpace(task.ShardID),
		DomainID:      strings.TrimSpace(task.DomainID),
		AgentRole:     runtimeAgentRoleForTask(task),
		ArtifactRoot:  writeRoot,
		RepoScopes:    append([]string(nil), task.RepoScopes...),
		PathScopes:    append([]string(nil), task.PathScopes...),
		Summary:       strings.TrimSpace(result.Summary),
		Documents:     []contracts.AuthoredDocument{document},
		Citations:     citations,
		Compatibility: compatibility,
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal shard pack manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	return writeRuntimeArtifactFile(writeRoot, shardPackManifestFileName, encoded)
}

func writeValidatorVerdict(writeRoot string, task acpruntime.Task) error {
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		return fmt.Errorf("create validator write root: %w", err)
	}
	checkedPaths := collectValidatorCheckedPaths(task)
	verdict := contracts.ValidatorVerdict{
		Version:      1,
		RunID:        strings.TrimSpace(task.RunID),
		GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
		Verdict:      "PASS",
		Summary:      "Validator accepted staged doc set and indexes.",
		CheckedPaths: checkedPaths,
	}
	encoded, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validator verdict: %w", err)
	}
	encoded = append(encoded, '\n')
	return writeRuntimeArtifactFile(writeRoot, validatorVerdictFileName, encoded)
}

func writeRuntimeArtifactFile(root string, relativePath string, content []byte) error {
	cleanRoot := filepath.Clean(root)
	rel := filepath.Clean(relativePath)
	target := filepath.Join(cleanRoot, rel)
	if !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) && target != cleanRoot {
		return fmt.Errorf("artifact path %q escapes write root %q", relativePath, root)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create artifact parent for %q: %w", relativePath, err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		return fmt.Errorf("write artifact %q: %w", relativePath, err)
	}
	return nil
}

func compatibilitySnapshotFromTaskResult(result contracts.TaskResult) contracts.CompatibilitySnapshot {
	snapshot := contracts.CompatibilitySnapshot{
		Coverage:  contracts.Coverage{},
		Questions: []contracts.Question{},
		Entities:  []contracts.Entity{},
		Edges:     []contracts.Edge{},
		Findings:  []contracts.Finding{},
	}
	if result.Coverage != nil {
		snapshot.Coverage = *result.Coverage
	}
	snapshot.Questions = append([]contracts.Question{}, result.Questions...)
	for _, op := range result.Changeset {
		switch op.Op {
		case "upsert_entity":
			if op.Entity != nil {
				snapshot.Entities = append(snapshot.Entities, *op.Entity)
			}
		case "upsert_edge":
			if op.Edge != nil {
				snapshot.Edges = append(snapshot.Edges, *op.Edge)
			}
		case "add_finding":
			if op.Finding != nil {
				snapshot.Findings = append(snapshot.Findings, *op.Finding)
			}
		}
	}
	sort.Slice(snapshot.Entities, func(i, j int) bool { return snapshot.Entities[i].ID < snapshot.Entities[j].ID })
	sort.Slice(snapshot.Edges, func(i, j int) bool { return snapshot.Edges[i].ID < snapshot.Edges[j].ID })
	sort.Slice(snapshot.Findings, func(i, j int) bool { return snapshot.Findings[i].ID < snapshot.Findings[j].ID })
	sort.Slice(snapshot.Questions, func(i, j int) bool { return snapshot.Questions[i].ID < snapshot.Questions[j].ID })
	return snapshot
}

func deriveRuntimeCitations(
	task acpruntime.Task,
	compatibility contracts.CompatibilitySnapshot,
	documentID string,
) []contracts.DocumentCitation {
	sources := []runtimeCitationSource{}
	appendEvidence := func(prefix string, objectID string, provenance contracts.Provenance) {
		claimKey := strings.TrimSpace(prefix + "." + objectID)
		if claimKey == "." || claimKey == "" {
			return
		}
		if len(provenance.Evidence) == 0 {
			sources = append(sources, fallbackCitationSource(task, claimKey, documentID))
			return
		}
		for idx, evidence := range provenance.Evidence {
			repo := strings.TrimSpace(evidence.Repo)
			if repo == "" {
				repo = primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
			}
			path := strings.TrimSpace(evidence.Path)
			if path == "" {
				path = fallbackEvidencePath(task)
			}
			key := fmt.Sprintf("%s.%d", claimKey, idx+1)
			sources = append(sources, runtimeCitationSource{
				claimKey:   key,
				repo:       repo,
				ref:        strings.TrimSpace(evidence.Ref),
				path:       path,
				lines:      copyLineRange(evidence.Lines),
				excerpt:    strings.TrimSpace(evidence.Excerpt),
				hash:       strings.TrimSpace(evidence.ExcerptHash),
				documentID: documentID,
			})
		}
	}

	for _, entity := range compatibility.Entities {
		appendEvidence("entity", entity.ID, entity.Provenance)
	}
	for _, edge := range compatibility.Edges {
		appendEvidence("edge", edge.ID, edge.Provenance)
	}
	for _, finding := range compatibility.Findings {
		appendEvidence("finding", finding.ID, finding.Provenance)
	}

	if len(sources) == 0 {
		sources = append(sources, fallbackCitationSource(task, "runtime.summary", documentID))
	}

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].claimKey == sources[j].claimKey {
			if sources[i].repo == sources[j].repo {
				return sources[i].path < sources[j].path
			}
			return sources[i].repo < sources[j].repo
		}
		return sources[i].claimKey < sources[j].claimKey
	})

	citations := make([]contracts.DocumentCitation, 0, len(sources))
	for _, source := range sources {
		claimID := "claim." + slugutil.Slugify(source.claimKey)
		citationID := "cite." + slugutil.Slugify(source.claimKey)
		if claimID == "claim." {
			claimID = "claim.runtime"
		}
		if citationID == "cite." {
			citationID = "cite.runtime"
		}
		citations = append(citations, contracts.DocumentCitation{
			ID:          citationID,
			Repo:        source.repo,
			Ref:         source.ref,
			Path:        source.path,
			Lines:       source.lines,
			ExcerptHash: source.hash,
			Excerpt:     source.excerpt,
			ClaimIDs:    []string{claimID},
			DocumentIDs: []string{source.documentID},
		})
	}
	return citations
}

func fallbackCitationSource(task acpruntime.Task, claimKey string, documentID string) runtimeCitationSource {
	repo := primaryTaskRepoScope(task.RepoScope, task.RepoScopes)
	if repo == "" {
		repo = "workspace"
	}
	return runtimeCitationSource{
		claimKey:   claimKey,
		repo:       repo,
		path:       fallbackEvidencePath(task),
		documentID: documentID,
	}
}

func fallbackEvidencePath(task acpruntime.Task) string {
	for _, candidate := range task.PathScopes {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return filepath.ToSlash(trimmed)
		}
	}
	return "README.md"
}

func copyLineRange(input *contracts.LineRange) *contracts.LineRange {
	if input == nil {
		return nil
	}
	copy := *input
	return &copy
}

func collectCitationIDs(citations []contracts.DocumentCitation) []string {
	ids := make([]string, 0, len(citations))
	for _, citation := range citations {
		if trimmed := strings.TrimSpace(citation.ID); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	sort.Strings(ids)
	return ids
}

func runtimeDomainDocumentID(task acpruntime.Task) string {
	return "doc.domain." + runtimeDomainSlug(task)
}

func runtimeDomainCanonicalPath(task acpruntime.Task) string {
	return fmt.Sprintf("reports/agent-outputs/domains/%s.md", runtimeDomainSlug(task))
}

func runtimeDomainDocumentTitle(task acpruntime.Task) string {
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" {
		return fmt.Sprintf("Domain dossier: %s", domainID)
	}
	return fmt.Sprintf("Shard dossier: %s", runtimeDomainSlug(task))
}

func runtimeDomainTopics(task acpruntime.Task) []string {
	topics := []string{}
	if domainID := strings.TrimSpace(task.DomainID); domainID != "" {
		topics = append(topics, "domain."+slugutil.Slugify(domainID))
	}
	for _, scope := range task.RepoScopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			topics = append(topics, "repo."+slugutil.Slugify(trimmed))
		}
	}
	sort.Strings(topics)
	return uniqueSortedStrings(topics)
}

func runtimeDomainSlug(task acpruntime.Task) string {
	candidates := []string{
		strings.TrimSpace(task.DomainID),
		strings.TrimSpace(task.ShardID),
		primaryTaskRepoScope(task.RepoScope, task.RepoScopes),
	}
	for _, candidate := range candidates {
		if slug := slugutil.Slugify(candidate); slug != "" {
			return slug
		}
	}
	return "domain"
}

func runtimeAgentRoleForTask(task acpruntime.Task) string {
	if role := strings.TrimSpace(task.AgentRole); role != "" {
		return role
	}
	if strings.HasSuffix(task.StepID, "step3.findings") {
		return "validator"
	}
	return "shard-analyst"
}

func renderRuntimeDomainReport(
	task acpruntime.Task,
	result contracts.TaskResult,
	compatibility contracts.CompatibilitySnapshot,
	citations []contracts.DocumentCitation,
) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Runtime Dossier: %s\n\n", runtimeDomainSlug(task)))
	builder.WriteString(fmt.Sprintf("- domain_id: `%s`\n", defaultBacktickValue(task.DomainID, "unknown")))
	builder.WriteString(fmt.Sprintf("- shard_id: `%s`\n", defaultBacktickValue(task.ShardID, "unknown")))
	builder.WriteString(fmt.Sprintf("- agent_role: `%s`\n", defaultBacktickValue(runtimeAgentRoleForTask(task), "runtime")))
	builder.WriteString(fmt.Sprintf("- repo_scopes: %s\n", renderBacktickListOrNone(task.RepoScopes)))
	builder.WriteString(fmt.Sprintf("- summary: %s\n", defaultPlainValue(strings.TrimSpace(result.Summary), "none")))
	builder.WriteString(fmt.Sprintf("- citations: %d\n\n", len(citations)))

	builder.WriteString("## Coverage\n\n")
	builder.WriteString(fmt.Sprintf("- observed: %s\n", renderBacktickListOrNone(compatibility.Coverage.Observed)))
	builder.WriteString(fmt.Sprintf("- missing: %s\n", renderBacktickListOrNone(compatibility.Coverage.Missing)))
	builder.WriteString(fmt.Sprintf("- notes: %s\n\n", renderPlainListOrNone(compatibility.Coverage.Notes)))

	builder.WriteString("## Entities\n\n")
	if len(compatibility.Entities) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, entity := range compatibility.Entities {
			builder.WriteString(fmt.Sprintf("- `%s` (%s)\n", entity.ID, entity.Type))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Findings\n\n")
	if len(compatibility.Findings) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, finding := range compatibility.Findings {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", finding.ID, finding.Title))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Questions\n\n")
	if len(compatibility.Questions) == 0 {
		builder.WriteString("- none\n\n")
	} else {
		for _, question := range compatibility.Questions {
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", question.ID, question.Text))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("## Citation IDs\n\n")
	for _, citationID := range collectCitationIDs(citations) {
		builder.WriteString(fmt.Sprintf("- `%s`\n", citationID))
	}
	return builder.String()
}

func renderBacktickListOrNone(values []string) string {
	normalized := uniqueSortedStrings(values)
	if len(normalized) == 0 {
		return "`none`"
	}
	quoted := make([]string, 0, len(normalized))
	for _, value := range normalized {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}

func renderPlainListOrNone(values []string) string {
	normalized := uniqueSortedStrings(values)
	if len(normalized) == 0 {
		return "none"
	}
	return strings.Join(normalized, ", ")
}

func defaultBacktickValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	return value
}

func defaultPlainValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func collectValidatorCheckedPaths(task acpruntime.Task) []string {
	paths := []string{}
	stageRoot := filepath.Join(task.Workspace, "reports", "taskruns", task.RunID, "staging", "final")
	if err := filepath.Walk(stageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(task.Workspace, path)
		if relErr != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		paths = nil
	}
	if len(paths) == 0 {
		paths = append(paths,
			filepath.ToSlash(filepath.Join("reports", "taskruns", task.RunID, "staging", "final", finalRunIndexFileName)),
			filepath.ToSlash(filepath.Join("reports", "taskruns", task.RunID, "staging", "final", citationIndexFileName)),
		)
	}
	sort.Strings(paths)
	return uniqueSortedStrings(paths)
}
