package providercommon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

const maxRuntimeRecoveredCollectDocs = 12

var markdownSignalPattern = regexp.MustCompile("`([^`]+)`|\\*\\*([^*]+)\\*\\*")

type collectManifestRuntimeRecoveryReport struct {
	DocumentCount int
	EntityCount   int
	EdgeCount     int
	EvidencePath  string
}

type collectManifestRuntimeRecoveryDoc struct {
	RelPath string
	Title   string
	Text    string
	Terms   []string
}

func recoverCollectManifestFromAuthoredDocs(task acpruntime.Task, cause error) (collectManifestRuntimeRecoveryReport, error) {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return collectManifestRuntimeRecoveryReport{}, fmt.Errorf("collect manifest runtime recovery requires write_root")
	}
	docs, err := collectRuntimeRecoveryDocs(writeRoot)
	if err != nil {
		return collectManifestRuntimeRecoveryReport{}, err
	}
	if len(docs) == 0 {
		return collectManifestRuntimeRecoveryReport{}, fmt.Errorf("collect manifest runtime recovery found no non-bootstrap authored markdown docs")
	}

	repo := collectManifestRecoveryRepo(task)
	evidencePath := collectManifestRecoveryEvidencePath(task, docs)
	manifest := buildRecoveredCollectManifest(task, docs, repo, evidencePath, cause)
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return collectManifestRuntimeRecoveryReport{}, fmt.Errorf("encode recovered collect manifest: %w", err)
	}
	raw = append(raw, '\n')
	manifestPath := filepath.Join(filepath.Clean(writeRoot), ShardPackManifestFileName)
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		return collectManifestRuntimeRecoveryReport{}, fmt.Errorf("write recovered collect manifest: %w", err)
	}
	if err := artifactquality.ValidateCollectManifestInRoot(writeRoot); err != nil {
		return collectManifestRuntimeRecoveryReport{}, fmt.Errorf("validate recovered collect manifest: %w", err)
	}
	return collectManifestRuntimeRecoveryReport{
		DocumentCount: len(manifest.Documents),
		EntityCount:   len(manifest.Semantic.Entities),
		EdgeCount:     len(manifest.Semantic.Edges),
		EvidencePath:  evidencePath,
	}, nil
}

func collectRuntimeRecoveryDocs(writeRoot string) ([]collectManifestRuntimeRecoveryDoc, error) {
	root := filepath.Clean(writeRoot)
	docs := []collectManifestRuntimeRecoveryDoc{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil {
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			switch name {
			case ".git", ".hg", ".svn", "__pycache__", "node_modules":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if name == ShardPackManifestFileName || name == "runtime-execution.json" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasPrefix(rel, "../") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := strings.TrimSpace(string(raw))
		if text == "" || artifactquality.CollectDocumentBootstrapOnly(text) {
			return nil
		}
		docs = append(docs, collectManifestRuntimeRecoveryDoc{
			RelPath: rel,
			Title:   collectRuntimeRecoveryDocTitle(rel, text),
			Text:    text,
			Terms:   collectRuntimeRecoveryTerms(rel, text),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan collect authored docs: %w", err)
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].RelPath < docs[j].RelPath
	})
	if len(docs) > maxRuntimeRecoveredCollectDocs {
		docs = docs[:maxRuntimeRecoveredCollectDocs]
	}
	return docs, nil
}

func buildRecoveredCollectManifest(task acpruntime.Task, docs []collectManifestRuntimeRecoveryDoc, repo string, evidencePath string, cause error) contracts.ShardPackManifest {
	runID := firstNonEmptyString(task.RunID, "run-1")
	stepID := firstNonEmptyString(task.StepID, "init.step1.collect")
	shardSlug := slugComponent(firstNonEmptyString(task.ShardID, task.DomainID, strings.Join(task.PathScopes, "-"), "shard"))
	if shardSlug == "" {
		shardSlug = "shard"
	}
	shardID := firstNonEmptyString(task.ShardID, shardSlug)
	domainID := strings.TrimSpace(task.DomainID)
	artifactRoot := firstNonEmptyString(task.ArtifactRoot, fmt.Sprintf("reports/taskruns/%s/staging/shards/%s", runID, shardSlug))
	topic := firstNonEmptyString(slugComponent(domainID), shardSlug, "architecture")

	documents := make([]contracts.AuthoredDocument, 0, len(docs))
	citations := make([]contracts.DocumentCitation, 0, len(docs))
	for idx, doc := range docs {
		docSlug := slugComponent(strings.TrimSuffix(doc.RelPath, filepath.Ext(doc.RelPath)))
		if docSlug == "" {
			docSlug = fmt.Sprintf("doc-%d", idx+1)
		}
		docID := fmt.Sprintf("doc.%s.%s", idComponent(shardSlug), idComponent(docSlug))
		citationID := fmt.Sprintf("cite.%s.%s", idComponent(shardSlug), idComponent(docSlug))
		documents = append(documents, contracts.AuthoredDocument{
			ID:            docID,
			Kind:          "report",
			Title:         doc.Title,
			Path:          doc.RelPath,
			CanonicalPath: fmt.Sprintf("reports/as-is/%s/%s.md", shardSlug, docSlug),
			Topics:        []string{topic},
			CitationIDs:   []string{citationID},
		})
		citations = append(citations, contracts.DocumentCitation{
			ID:          citationID,
			Repo:        repo,
			Path:        evidencePath,
			ClaimIDs:    []string{fmt.Sprintf("claim.%s.%s.runtime_recovery", idComponent(shardSlug), idComponent(docSlug))},
			DocumentIDs: []string{docID},
		})
	}

	return contracts.ShardPackManifest{
		Version:      1,
		RunID:        runID,
		StepID:       stepID,
		ShardID:      shardID,
		DomainID:     domainID,
		AgentRole:    firstNonEmptyString(task.AgentRole, "shard-analyst"),
		ArtifactRoot: artifactRoot,
		RepoScopes:   nonEmptyStringList(append([]string{task.RepoScope}, task.RepoScopes...)),
		PathScopes:   nonEmptyStringList(task.PathScopes),
		Summary:      "Runtime recovered shard-pack-manifest.json from provider-authored collect documents after manifest-only repair did not complete.",
		Documents:    documents,
		Citations:    citations,
		Semantic:     recoveredCollectSemantic(task, docs, repo, evidencePath, shardSlug, topic, cause),
	}
}

func recoveredCollectSemantic(task acpruntime.Task, docs []collectManifestRuntimeRecoveryDoc, repo string, evidencePath string, shardSlug string, topic string, cause error) contracts.SemanticSnapshot {
	repoStem := idComponent(firstNonEmptyString(repo, "repo"))
	shardStem := idComponent(firstNonEmptyString(shardSlug, topic, "shard"))
	repoEntityID := "svc." + repoStem
	shardEntityID := "cmp." + shardStem
	evidence := []contracts.Evidence{{Repo: repo, Path: evidencePath}}
	entities := []contracts.Entity{
		{
			ID:   repoEntityID,
			Type: "service",
			Name: titleFromSlug(repoStem),
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.6,
				Evidence:   evidence,
			},
		},
		{
			ID:   shardEntityID,
			Type: "component",
			Name: titleFromSlug(shardSlug),
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.55,
				Evidence:   evidence,
			},
		},
	}
	edges := []contracts.Edge{{
		ID:   fmt.Sprintf("edge.%s.runtime-recovery.documents", shardStem),
		Type: "documents",
		From: repoEntityID,
		To:   shardEntityID,
		Name: "documents provider-authored collect surface",
		Provenance: contracts.Provenance{
			Kind:       "observation",
			Confidence: 0.55,
			Evidence:   evidence,
		},
	}}

	observed := []string{topic, "provider-authored collect document"}
	seenTerms := map[string]struct{}{}
	termEntityIDs := []string{}
	termNames := []string{}
	for _, doc := range docs {
		docStem := idComponent(strings.TrimSuffix(doc.RelPath, filepath.Ext(doc.RelPath)))
		docEntityID := "doc." + shardStem + "." + docStem
		entities = append(entities, contracts.Entity{
			ID:   docEntityID,
			Type: "document",
			Name: doc.Title,
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.65,
				Evidence:   evidence,
			},
		})
		edges = append(edges, contracts.Edge{
			ID:   "edge." + shardStem + "." + docStem + ".describes",
			Type: "documents",
			From: docEntityID,
			To:   shardEntityID,
			Name: "describes recovered collect scope",
			Provenance: contracts.Provenance{
				Kind:       "observation",
				Confidence: 0.55,
				Evidence:   evidence,
			},
		})
		for _, term := range doc.Terms {
			key := slugComponent(term)
			if key == "" {
				continue
			}
			if _, exists := seenTerms[key]; exists {
				continue
			}
			seenTerms[key] = struct{}{}
			observed = append(observed, term)
			if len(entities) >= 14 {
				continue
			}
			entityType := collectRuntimeRecoveryEntityType(term)
			termEntityID := collectRuntimeRecoveryEntityID(entityType, shardStem, key)
			entities = append(entities, contracts.Entity{
				ID:   termEntityID,
				Type: entityType,
				Name: collectRuntimeRecoveryEntityName(term, key),
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.58,
					Evidence:   evidence,
				},
			})
			termEntityIDs = append(termEntityIDs, termEntityID)
			termNames = append(termNames, collectRuntimeRecoveryEntityName(term, key))
			edgeType := collectRuntimeRecoveryEdgeType(term, entityType)
			edges = append(edges, contracts.Edge{
				ID:   "edge." + shardStem + "." + idComponent(edgeType) + "." + idComponent(key),
				Type: edgeType,
				From: shardEntityID,
				To:   termEntityID,
				Name: fmt.Sprintf("%s %s %s", titleFromSlug(shardSlug), strings.ReplaceAll(edgeType, "_", " "), collectRuntimeRecoveryEntityName(term, key)),
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.56,
					Evidence:   evidence,
				},
			})
			edges = append(edges, contracts.Edge{
				ID:   "edge." + shardStem + "." + idComponent(key) + ".mentioned-by-doc",
				Type: "documents",
				From: docEntityID,
				To:   termEntityID,
				Name: "mentions recovered collect concept",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.5,
					Evidence:   evidence,
				},
			})
		}
	}
	if len(termEntityIDs) == 0 {
		for _, fallback := range collectRuntimeRecoveryFallbackTerms(task, shardSlug, topic) {
			key := slugComponent(fallback)
			if key == "" {
				continue
			}
			if _, exists := seenTerms[key]; exists {
				continue
			}
			seenTerms[key] = struct{}{}
			entityType := collectRuntimeRecoveryEntityType(fallback)
			termEntityID := collectRuntimeRecoveryEntityID(entityType, shardStem, key)
			entityName := collectRuntimeRecoveryEntityName(fallback, key)
			entities = append(entities, contracts.Entity{
				ID:   termEntityID,
				Type: entityType,
				Name: entityName,
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.48,
					Evidence:   evidence,
				},
			})
			termEntityIDs = append(termEntityIDs, termEntityID)
			termNames = append(termNames, entityName)
			edgeType := collectRuntimeRecoveryEdgeType(fallback, entityType)
			edges = append(edges, contracts.Edge{
				ID:   "edge." + shardStem + "." + idComponent(edgeType) + "." + idComponent(key),
				Type: edgeType,
				From: shardEntityID,
				To:   termEntityID,
				Name: fmt.Sprintf("%s %s %s", titleFromSlug(shardSlug), strings.ReplaceAll(edgeType, "_", " "), entityName),
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.48,
					Evidence:   evidence,
				},
			})
			observed = append(observed, fallback)
			break
		}
	}

	missing := []string{"provider did not complete shard-pack-manifest.json before runtime contract recovery"}
	if cause != nil {
		missing = append(missing, "manifest-only provider repair failure: "+compactRecoveryCause(cause))
	}
	observedNames := "the provider-authored collect documents"
	if len(termNames) > 0 {
		observedNames = strings.Join(termNames[:minInt(len(termNames), 6)], ", ")
	}
	questionIDStem := idComponent(firstNonEmptyString(shardSlug, "collect-recovery"))
	return contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{
			Observed: dedupeStrings(observed),
			Missing:  dedupeStrings(missing),
			Notes: []string{
				"Runtime recovery is limited to manifest reconstruction from provider-authored markdown and bounded scoped evidence.",
				"Downstream quality gates still decide whether recovered artifacts are complete enough for acceptance.",
			},
		},
		Questions: []contracts.Question{{
			ID:         "question." + questionIDStem + ".manifest_recovery_followup",
			Text:       fmt.Sprintf("Should this shard be rerun for richer provider-authored manifest semantics around %s, or is the recovered manifest sufficient for the current diagnostic analysis?", observedNames),
			Priority:   "medium",
			RelatedIDs: []string{shardEntityID},
		}},
		Entities: entities,
		Edges:    edges,
		Findings: []contracts.Finding{{
			ID:          "finding." + questionIDStem + ".manifest_recovery_applied",
			Severity:    "medium",
			Title:       "Collect manifest recovered from authored documents",
			Description: fmt.Sprintf("The provider wrote collect markdown that identified %s but did not complete shard-pack-manifest.json during the primary or manifest-only repair process. Runtime recovery reconstructed the manifest from the provider-authored documents so downstream validation can surface remaining quality gaps instead of accepting an empty shard.", observedNames),
			RuleID:      "rule.collect_manifest.runtime_recovery",
			RelatedIDs:  append([]string{shardEntityID}, termEntityIDs[:minInt(len(termEntityIDs), 4)]...),
			Provenance: contracts.Provenance{
				Kind:       "inference",
				Confidence: 0.55,
				Evidence:   evidence,
			},
		}},
	}
}

func collectRuntimeRecoveryEntityType(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if containsAnySubstring(lower, "postgres", "redis", "clickhouse", "kafka", "redpanda", "minio", "elasticsearch", "database", " db", "db ") {
		return "datastore"
	}
	if containsAnySubstring(lower, "service", "worker", "api", "ingestion", "capture", "web", "temporal", "celery", "cli", "parser", "runtime", "sdk") {
		return "service"
	}
	return "component"
}

func collectRuntimeRecoveryEdgeType(name string, entityType string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if entityType == "datastore" {
		return "depends_on"
	}
	if containsAnySubstring(lower, "docker", "compose", "turbo", "pnpm", "uv", "pytest", "ruff", "mypy", "tsconfig", "package.json", "pyproject", "env", "makefile", "config", "deploy") {
		return "configures"
	}
	return "uses"
}

func collectRuntimeRecoveryEntityID(entityType string, shardStem string, key string) string {
	prefix := "cmp"
	switch entityType {
	case "datastore":
		prefix = "ds"
	case "service":
		prefix = "svc"
	}
	return prefix + "." + shardStem + "." + idComponent(key)
}

func collectRuntimeRecoveryEntityName(term string, key string) string {
	term = strings.TrimSpace(term)
	if term != "" {
		return term
	}
	return titleFromSlug(key)
}

func collectRuntimeRecoveryFallbackTerms(task acpruntime.Task, shardSlug string, topic string) []string {
	values := []string{}
	values = append(values, task.DomainID, task.ShardID, topic, shardSlug)
	values = append(values, task.PathScopes...)
	for _, value := range values {
		if term := normalizeRecoveryTerm(value); term != "" {
			return []string{term}
		}
	}
	return nil
}

func containsAnySubstring(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func collectRuntimeRecoveryDocTitle(relPath string, text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if title != "" {
				return title
			}
		}
	}
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	return titleFromSlug(base)
}

func collectRuntimeRecoveryTerms(relPath string, text string) []string {
	candidates := []string{}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			candidates = append(candidates, strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
		}
		if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
			body := strings.TrimSpace(strings.TrimLeft(trimmed, "-* "))
			if before, _, ok := strings.Cut(body, ":"); ok {
				candidates = append(candidates, before)
			}
		}
		for _, match := range markdownSignalPattern.FindAllStringSubmatch(trimmed, -1) {
			for _, value := range match[1:] {
				if value != "" {
					candidates = append(candidates, value)
				}
			}
		}
	}
	candidates = append(candidates, strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)))

	terms := []string{}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		term := normalizeRecoveryTerm(candidate)
		if term == "" {
			continue
		}
		key := strings.ToLower(term)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		terms = append(terms, term)
		if len(terms) >= 8 {
			break
		}
	}
	return terms
}

func normalizeRecoveryTerm(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`*_#[](){}:;,.\"'")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 72 {
		value = strings.TrimSpace(value[:72])
	}
	if value == "" {
		return ""
	}
	normalized := strings.ToLower(value)
	if strings.Contains(normalized, "/") || strings.Contains(normalized, "\\") {
		return ""
	}
	stop := map[string]struct{}{
		"architecture": {}, "architectural overview": {}, "collect": {}, "collect overview": {},
		"coverage": {}, "evidence": {}, "findings": {}, "follow-up": {}, "overview": {},
		"questions": {}, "remaining questions": {}, "repository": {}, "scope": {}, "summary": {},
	}
	if _, blocked := stop[normalized]; blocked {
		return ""
	}
	if slugComponent(value) == "" {
		return ""
	}
	return value
}

func collectManifestRecoveryRepo(task acpruntime.Task) string {
	if value := strings.TrimSpace(task.RepoScope); value != "" {
		return value
	}
	for _, value := range task.RepoScopes {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "repo"
}

func collectManifestRecoveryEvidencePath(task acpruntime.Task, docs []collectManifestRuntimeRecoveryDoc) string {
	pathScopes := []string{}
	for _, value := range task.PathScopes {
		value = cleanRecoveryEvidencePath(value)
		if value != "" {
			pathScopes = append(pathScopes, value)
		}
	}
	candidates := preferredRecoveryEvidenceCandidates(pathScopes)
	for _, value := range []string{"README.md", "README.adoc", "README.rst", "package.json", "pyproject.toml", "go.mod", "Makefile", "docker-compose.yml"} {
		candidates = append(candidates, value)
	}
	for _, candidate := range candidates {
		if recoveryEvidencePathExists(task.ReadContextRoots, candidate) {
			return candidate
		}
	}
	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}
	if len(docs) > 0 {
		return docs[0].RelPath
	}
	return "README.md"
}

func preferredRecoveryEvidenceCandidates(paths []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	add := func(value string) {
		value = cleanRecoveryEvidencePath(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	preferred := []func(string) bool{
		func(value string) bool {
			lower := strings.ToLower(value)
			return lower == "readme" || strings.HasPrefix(lower, "readme.")
		},
		func(value string) bool {
			return strings.EqualFold(value, "makefile")
		},
		func(value string) bool {
			lower := strings.ToLower(value)
			switch lower {
			case "pom.xml", "package.json", "go.mod", "build.gradle", "build.gradle.kts", "gradlew", "mvnw", "dockerfile", "justfile":
				return true
			default:
				return strings.HasPrefix(lower, "skaffold") || strings.HasPrefix(lower, "docker-compose")
			}
		},
	}
	for _, match := range preferred {
		for _, path := range paths {
			if match(path) {
				add(path)
			}
		}
	}
	for _, path := range paths {
		add(path)
	}
	return result
}

func cleanRecoveryEvidencePath(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "./")
	value = strings.Trim(value, "/")
	if value == "" || value == "." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") {
		return ""
	}
	return value
}

func recoveryEvidencePathExists(roots []string, rel string) bool {
	rel = cleanRecoveryEvidencePath(rel)
	if rel == "" {
		return false
	}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(filepath.Clean(root), filepath.FromSlash(rel)))
		if err == nil && info != nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func compactRecoveryCause(err error) string {
	if err == nil {
		return ""
	}
	value := strings.Join(strings.Fields(err.Error()), " ")
	if len(value) <= 180 {
		return value
	}
	return value[:177] + "..."
}

func slugComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if len(result) > 72 {
		result = strings.Trim(result[:72], "-")
	}
	return result
}

func idComponent(value string) string {
	value = strings.ReplaceAll(slugComponent(value), "-", ".")
	if value == "" {
		return "shard"
	}
	return value
}

func titleFromSlug(value string) string {
	parts := strings.Fields(strings.ReplaceAll(slugComponent(value), "-", " "))
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return "Shard Overview"
	}
	return strings.Join(parts, " ")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonEmptyStringList(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
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

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
