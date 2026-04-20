package artifactquality

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

func EnsureCanonicalCollectManifest(task acpruntime.Task, result contracts.TaskResult) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return nil
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			manifest, synthErr := synthesizeCollectManifest(task, result)
			if synthErr != nil {
				return synthErr
			}
			encoded, marshalErr := json.MarshalIndent(manifest, "", "  ")
			if marshalErr != nil {
				return fmt.Errorf("marshal synthesized shard pack manifest: %w", marshalErr)
			}
			encoded = append(encoded, '\n')
			if _, parseErr := contracts.ParseShardPackManifest(encoded); parseErr != nil {
				return parseErr
			}
			return os.WriteFile(manifestPath, encoded, 0o644)
		}
		return err
	}

	manifest, changed, err := canonicalizeCollectManifest(raw, task, result)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal canonical shard pack manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return err
	}
	return os.WriteFile(manifestPath, encoded, 0o644)
}

func canonicalizeCollectManifest(raw []byte, task acpruntime.Task, result contracts.TaskResult) (contracts.ShardPackManifest, bool, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		manifest, synthErr := synthesizeCollectManifest(task, result)
		if synthErr != nil {
			return contracts.ShardPackManifest{}, false, fmt.Errorf("decode shard pack manifest for canonicalization: %w", err)
		}
		return manifest, true, nil
	}

	manifest := defaultCollectManifestSkeleton(task, result)
	mergeLooseManifestMetadata(&manifest, payload, task, result)
	manifest.Documents = parseLooseManifestDocuments(payload["documents"])
	manifest.Citations = parseLooseManifestCitations(payload["citations"])
	if len(manifest.Documents) == 0 {
		manifest.Documents = extractTaskResultDocArtifacts(task, result)
	}
	if len(manifest.Documents) == 0 {
		fallbackDocs, fallbackErr := discoverWriteRootDocs(task)
		if fallbackErr != nil {
			return contracts.ShardPackManifest{}, false, fallbackErr
		}
		manifest.Documents = fallbackDocs
	}

	canonicalizeManifestDocuments(&manifest, task)
	canonicalizeManifestCitations(&manifest, task)
	linkManifestDocumentsAndCitations(&manifest, task)
	manifest.Compatibility = CompatibilitySnapshotFromTaskResult(result)

	encoded, err := json.Marshal(manifest)
	if err != nil {
		return contracts.ShardPackManifest{}, false, fmt.Errorf("marshal canonical shard pack manifest candidate: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		manifest, synthErr := synthesizeCollectManifest(task, result)
		if synthErr != nil {
			return contracts.ShardPackManifest{}, false, err
		}
		return manifest, true, nil
	}

	var before any
	var after any
	changed := true
	if json.Unmarshal(raw, &before) == nil && json.Unmarshal(encoded, &after) == nil && reflect.DeepEqual(before, after) {
		changed = false
	}
	return manifest, changed, nil
}

func synthesizeCollectManifest(task acpruntime.Task, result contracts.TaskResult) (contracts.ShardPackManifest, error) {
	manifest := defaultCollectManifestSkeleton(task, result)
	manifest.Documents = extractTaskResultDocArtifacts(task, result)
	if len(manifest.Documents) == 0 {
		discovered, err := discoverWriteRootDocs(task)
		if err != nil {
			return contracts.ShardPackManifest{}, err
		}
		manifest.Documents = discovered
	}
	canonicalizeManifestDocuments(&manifest, task)
	canonicalizeManifestCitations(&manifest, task)
	linkManifestDocumentsAndCitations(&manifest, task)

	encoded, err := json.Marshal(manifest)
	if err != nil {
		return contracts.ShardPackManifest{}, fmt.Errorf("marshal synthesized shard pack manifest: %w", err)
	}
	if _, err := contracts.ParseShardPackManifest(encoded); err != nil {
		return contracts.ShardPackManifest{}, err
	}
	return manifest, nil
}

func defaultCollectManifestSkeleton(task acpruntime.Task, result contracts.TaskResult) contracts.ShardPackManifest {
	runID := firstNonEmpty(strings.TrimSpace(task.RunID), strings.TrimSpace(result.Meta.RunID))
	stepID := firstNonEmpty(strings.TrimSpace(task.StepID), strings.TrimSpace(result.Meta.StepID))
	shardID := firstNonEmpty(strings.TrimSpace(task.ShardID), strings.TrimSpace(result.Meta.ShardID), "shard")
	artifactRoot := normalizeManifestArtifactRoot(
		strings.TrimSpace(task.ArtifactRoot),
		strings.TrimSpace(task.ArtifactRoot),
		strings.TrimSpace(task.Workspace),
		strings.TrimSpace(task.WriteRoot),
		runID,
		shardID,
	)
	repoScopes := normalizeOrderedUniqueStrings(append([]string{}, append(task.RepoScopes, result.Meta.RepoScopes...)...))
	pathScopes := normalizeOrderedUniqueStrings(append([]string{}, append(task.PathScopes, result.Meta.PathScopes...)...))
	return contracts.ShardPackManifest{
		Version:       1,
		RunID:         runID,
		StepID:        stepID,
		ShardID:       shardID,
		DomainID:      firstNonEmpty(strings.TrimSpace(task.DomainID)),
		AgentRole:     canonicalAgentRole(task),
		ArtifactRoot:  artifactRoot,
		RepoScopes:    repoScopes,
		PathScopes:    pathScopes,
		Summary:       firstNonEmpty(strings.TrimSpace(result.Summary)),
		Documents:     []contracts.AuthoredDocument{},
		Citations:     []contracts.DocumentCitation{},
		Compatibility: CompatibilitySnapshotFromTaskResult(result),
	}
}

func mergeLooseManifestMetadata(manifest *contracts.ShardPackManifest, payload map[string]any, task acpruntime.Task, result contracts.TaskResult) {
	if manifest == nil {
		return
	}
	if version, ok := intValue(payload["version"]); ok && version > 0 {
		manifest.Version = version
	}
	manifest.RunID = firstNonEmpty(strings.TrimSpace(task.RunID), looseString(payload["run_id"]), strings.TrimSpace(result.Meta.RunID), manifest.RunID)
	manifest.StepID = firstNonEmpty(strings.TrimSpace(task.StepID), looseString(payload["step_id"]), strings.TrimSpace(result.Meta.StepID), manifest.StepID)
	manifest.ShardID = firstNonEmpty(strings.TrimSpace(task.ShardID), looseString(payload["shard_id"]), strings.TrimSpace(result.Meta.ShardID), manifest.ShardID)
	manifest.DomainID = firstNonEmpty(strings.TrimSpace(task.DomainID), looseString(payload["domain_id"]), manifest.DomainID)
	manifest.AgentRole = firstNonEmpty(canonicalAgentRole(task), looseString(payload["agent_role"]), manifest.AgentRole)
	manifest.ArtifactRoot = normalizeManifestArtifactRoot(
		firstNonEmpty(strings.TrimSpace(task.ArtifactRoot), looseString(payload["artifact_root"]), manifest.ArtifactRoot),
		strings.TrimSpace(task.ArtifactRoot),
		strings.TrimSpace(task.Workspace),
		strings.TrimSpace(task.WriteRoot),
		firstNonEmpty(strings.TrimSpace(task.RunID), strings.TrimSpace(result.Meta.RunID), manifest.RunID),
		firstNonEmpty(strings.TrimSpace(task.ShardID), strings.TrimSpace(result.Meta.ShardID), manifest.ShardID),
	)
	manifest.RepoScopes = normalizeOrderedUniqueStrings(append(append([]string{}, looseStringSlice(payload["repo_scopes"])...), manifest.RepoScopes...))
	manifest.PathScopes = normalizeOrderedUniqueStrings(append(append([]string{}, looseStringSlice(payload["path_scopes"])...), manifest.PathScopes...))
	manifest.Summary = firstNonEmpty(strings.TrimSpace(result.Summary), looseString(payload["summary"]), manifest.Summary)
}

func parseLooseManifestDocuments(value any) []contracts.AuthoredDocument {
	items := asObjectSlice(value)
	documents := make([]contracts.AuthoredDocument, 0, len(items))
	for _, item := range items {
		doc := contracts.AuthoredDocument{
			ID:            looseString(item["id"]),
			Kind:          looseString(item["kind"]),
			Title:         looseString(item["title"]),
			Path:          looseString(item["path"]),
			CanonicalPath: looseString(item["canonical_path"]),
			Topics:        looseStringSlice(item["topics"]),
			CitationIDs:   looseStringSlice(item["citation_ids"]),
			Status:        looseString(item["status"]),
		}
		if len(doc.CitationIDs) == 0 {
			doc.CitationIDs = extractLegacyCitationIDs(item["citations"])
		}
		documents = append(documents, doc)
	}
	return documents
}

func parseLooseManifestCitations(value any) []contracts.DocumentCitation {
	items := asObjectSlice(value)
	citations := make([]contracts.DocumentCitation, 0, len(items))
	for _, item := range items {
		citation := contracts.DocumentCitation{
			ID:          looseString(item["id"]),
			Repo:        looseString(item["repo"]),
			Ref:         looseString(item["ref"]),
			Path:        looseString(item["path"]),
			ExcerptHash: looseString(item["excerpt_hash"]),
			Excerpt:     looseString(item["excerpt"]),
			ClaimIDs:    looseStringSlice(item["claim_ids"]),
			DocumentIDs: looseStringSlice(item["document_ids"]),
		}
		if linesMap, ok := item["lines"].(map[string]any); ok {
			start, startOK := intValue(linesMap["start"])
			end, endOK := intValue(linesMap["end"])
			if startOK && endOK && start > 0 && end > 0 {
				citation.Lines = &contracts.LineRange{Start: start, End: end}
			}
		}
		citations = append(citations, citation)
	}
	return citations
}

func extractTaskResultDocArtifacts(task acpruntime.Task, result contracts.TaskResult) []contracts.AuthoredDocument {
	documents := []contracts.AuthoredDocument{}
	for _, op := range result.Changeset {
		if op.Op != "add_doc_artifact" || op.DocArtifact == nil {
			continue
		}
		doc := contracts.AuthoredDocument{
			ID:            strings.TrimSpace(op.DocArtifact.ID),
			Kind:          strings.TrimSpace(op.DocArtifact.Kind),
			Title:         strings.TrimSpace(op.DocArtifact.Title),
			Path:          strings.TrimSpace(op.DocArtifact.Path),
			CanonicalPath: "",
			Topics:        normalizeOrderedUniqueStrings(op.DocArtifact.RelatedIDs),
			CitationIDs:   []string{},
		}
		if doc.ID == "" && doc.Path != "" {
			doc.ID = "doc." + slugutil.Slugify(strings.TrimSuffix(filepath.Base(doc.Path), filepath.Ext(doc.Path)))
		}
		doc.CanonicalPath = fallbackCanonicalPath(task, doc.Path, doc.ID)
		documents = append(documents, doc)
	}
	return documents
}

func discoverWriteRootDocs(task acpruntime.Task) ([]contracts.AuthoredDocument, error) {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return []contracts.AuthoredDocument{}, nil
	}
	paths := []string{}
	err := filepath.WalkDir(writeRoot, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(writeRoot, candidate)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == shardPackManifestFile {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(rel))
		switch ext {
		case ".md", ".markdown", ".txt", ".mmd", ".mermaid":
			paths = append(paths, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover authored documents in write_root: %w", err)
	}
	sort.Strings(paths)
	documents := make([]contracts.AuthoredDocument, 0, len(paths))
	for idx, rel := range paths {
		base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		slug := slugutil.Slugify(base)
		if slug == "" {
			slug = fmt.Sprintf("doc-%d", idx+1)
		}
		docID := fmt.Sprintf("doc.%s.%s", shardSlug(task), slug)
		documents = append(documents, contracts.AuthoredDocument{
			ID:            docID,
			Kind:          "report",
			Title:         titleFromPath(rel),
			Path:          rel,
			CanonicalPath: fallbackCanonicalPath(task, rel, docID),
			Topics:        []string{},
			CitationIDs:   []string{},
		})
	}
	return documents, nil
}

func canonicalizeManifestDocuments(manifest *contracts.ShardPackManifest, task acpruntime.Task) {
	if manifest == nil {
		return
	}
	seenIDs := map[string]int{}
	canonicalDocs := make([]contracts.AuthoredDocument, 0, len(manifest.Documents))
	for idx, document := range manifest.Documents {
		doc := document
		doc.Path = normalizeManifestDocumentPath(doc.Path, manifest.ArtifactRoot, task.WriteRoot, manifest.ShardID)
		if doc.Path == "" {
			doc.Path = fallbackDocumentPath(idx)
		}
		doc.ID = ensureUniqueDocID(firstNonEmpty(strings.TrimSpace(doc.ID), "doc."+shardSlug(task)+"."+slugFromPath(doc.Path), fmt.Sprintf("doc.%s.%d", shardSlug(task), idx+1)), seenIDs)
		doc.Kind = firstNonEmpty(strings.TrimSpace(doc.Kind), "report")
		doc.Title = firstNonEmpty(strings.TrimSpace(doc.Title), titleFromPath(doc.Path))
		doc.CanonicalPath = normalizeCanonicalPath(firstNonEmpty(strings.TrimSpace(doc.CanonicalPath), fallbackCanonicalPath(task, doc.Path, doc.ID)), task, doc)
		doc.Topics = normalizeOrderedUniqueStrings(doc.Topics)
		doc.CitationIDs = normalizeOrderedUniqueStrings(doc.CitationIDs)
		canonicalDocs = append(canonicalDocs, doc)
	}
	manifest.Documents = canonicalDocs
}

func canonicalizeManifestCitations(manifest *contracts.ShardPackManifest, task acpruntime.Task) {
	if manifest == nil {
		return
	}
	primaryRepo := primaryRepoScope(task, manifest.RepoScopes)
	defaultEvidencePath := fallbackEvidencePath(task)
	seenIDs := map[string]int{}
	seenClaimIDs := map[string]int{}
	canonical := make([]contracts.DocumentCitation, 0, len(manifest.Citations))
	for idx, citation := range manifest.Citations {
		item := citation
		baseID := firstNonEmpty(strings.TrimSpace(item.ID), fmt.Sprintf("cite.%s.%d", shardSlug(task), idx+1))
		item.ID = ensureUniqueCitationID(baseID, seenIDs)
		item.Repo = firstNonEmpty(strings.TrimSpace(item.Repo), primaryRepo)
		item.Path = firstNonEmpty(strings.TrimSpace(item.Path), defaultEvidencePath)
		item.ClaimIDs = normalizeOrderedUniqueStrings(item.ClaimIDs)
		if len(item.ClaimIDs) == 0 {
			item.ClaimIDs = []string{fmt.Sprintf("claim.%s.%d", shardSlug(task), idx+1)}
		}
		normalizedClaims := make([]string, 0, len(item.ClaimIDs))
		for _, claimID := range item.ClaimIDs {
			normalizedClaims = append(normalizedClaims, ensureUniqueClaimID(claimID, seenClaimIDs, task))
		}
		item.ClaimIDs = normalizedClaims
		item.DocumentIDs = normalizeOrderedUniqueStrings(item.DocumentIDs)
		canonical = append(canonical, item)
	}
	manifest.Citations = canonical
}

func linkManifestDocumentsAndCitations(manifest *contracts.ShardPackManifest, task acpruntime.Task) {
	if manifest == nil {
		return
	}
	if len(manifest.Documents) == 0 {
		manifest.Citations = []contracts.DocumentCitation{}
		return
	}
	if len(manifest.Citations) == 0 {
		manifest.Citations = append(manifest.Citations, contracts.DocumentCitation{
			ID:          fmt.Sprintf("cite.%s.fallback", shardSlug(task)),
			Repo:        primaryRepoScope(task, manifest.RepoScopes),
			Path:        fallbackEvidencePath(task),
			ClaimIDs:    []string{fmt.Sprintf("claim.%s.fallback", shardSlug(task))},
			DocumentIDs: []string{manifest.Documents[0].ID},
		})
	}

	knownDocs := map[string]struct{}{}
	for _, document := range manifest.Documents {
		knownDocs[document.ID] = struct{}{}
	}
	knownCitationIDs := map[string]struct{}{}
	for idx := range manifest.Citations {
		citation := &manifest.Citations[idx]
		filteredDocIDs := []string{}
		for _, docID := range citation.DocumentIDs {
			if _, ok := knownDocs[docID]; ok {
				filteredDocIDs = append(filteredDocIDs, docID)
			}
		}
		if len(filteredDocIDs) == 0 {
			filteredDocIDs = []string{manifest.Documents[0].ID}
		}
		citation.DocumentIDs = normalizeOrderedUniqueStrings(filteredDocIDs)
		knownCitationIDs[citation.ID] = struct{}{}
	}

	for idx := range manifest.Documents {
		document := &manifest.Documents[idx]
		filteredCitationIDs := []string{}
		for _, citationID := range document.CitationIDs {
			if _, ok := knownCitationIDs[citationID]; ok {
				filteredCitationIDs = append(filteredCitationIDs, citationID)
			}
		}
		if len(filteredCitationIDs) == 0 {
			filteredCitationIDs = []string{manifest.Citations[0].ID}
		}
		document.CitationIDs = normalizeOrderedUniqueStrings(filteredCitationIDs)
	}
}

func normalizeManifestArtifactRoot(
	rawValue string,
	taskArtifactRoot string,
	workspace string,
	writeRoot string,
	runID string,
	shardID string,
) string {
	candidate := strings.TrimSpace(rawValue)
	if candidate == "" {
		candidate = strings.TrimSpace(taskArtifactRoot)
	}
	if candidate == "" {
		candidate = deriveArtifactRootFromWorkspace(workspace, writeRoot)
		if candidate == "" {
			return "."
		}
	}

	candidate = filepath.ToSlash(filepath.Clean(candidate))
	if strings.HasPrefix(candidate, "/") {
		workspaceRoot := strings.TrimSpace(workspace)
		if workspaceRoot != "" {
			if rel, ok := pathRelativeToRoot(candidate, workspaceRoot); ok {
				candidate = rel
			} else {
				candidate = path.Base(candidate)
			}
		} else {
			candidate = path.Base(candidate)
		}
	}

	root := strings.TrimSpace(writeRoot)
	if root != "" {
		rootBase := strings.TrimSpace(path.Base(filepath.ToSlash(filepath.Clean(root))))
		if rootBase != "" {
			if candidate == rootBase {
				derived := deriveArtifactRootFromWorkspace(workspace, root)
				if derived != "" {
					candidate = derived
				}
			} else if strings.HasPrefix(candidate, rootBase+"/") {
				candidate = strings.TrimPrefix(candidate, rootBase+"/")
			}
		}
	}

	candidate = sanitizeRelativePath(candidate)
	if candidate == "" {
		derived := deriveArtifactRootFromWorkspace(workspace, writeRoot)
		if derived != "" {
			return derived
		}
		return "."
	}
	return candidate
}

func deriveArtifactRootFromWorkspace(workspace string, writeRoot string) string {
	workspace = strings.TrimSpace(workspace)
	writeRoot = strings.TrimSpace(writeRoot)
	if workspace == "" || writeRoot == "" {
		return ""
	}
	workspaceClean := filepath.Clean(workspace)
	writeRootClean := filepath.Clean(writeRoot)
	rel, err := filepath.Rel(workspaceClean, writeRootClean)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return ""
	}
	rel = sanitizeRelativePath(rel)
	if rel == "" {
		return "."
	}
	return rel
}

func pathRelativeToRoot(candidateAbs string, root string) (string, bool) {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidateAbs))
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	rel = sanitizeRelativePath(rel)
	if rel == "" {
		return ".", true
	}
	return rel, true
}

func normalizeManifestDocumentPath(rawPath string, artifactRoot string, writeRoot string, shardID string) string {
	value := strings.TrimSpace(rawPath)
	if value == "" {
		return ""
	}
	pathValue := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	writeRootValue := filepath.ToSlash(filepath.Clean(strings.TrimSpace(writeRoot)))
	if writeRootValue != "" && strings.HasPrefix(pathValue, writeRootValue+"/") {
		pathValue = strings.TrimPrefix(pathValue, writeRootValue+"/")
	}
	artifactRootValue := filepath.ToSlash(filepath.Clean(strings.TrimSpace(artifactRoot)))
	if artifactRootValue != "" && strings.HasPrefix(pathValue, artifactRootValue+"/") {
		pathValue = strings.TrimPrefix(pathValue, artifactRootValue+"/")
	}
	if strings.HasPrefix(pathValue, "reports/taskruns/") {
		pathValue = stripTaskrunsStagingPrefix(pathValue, artifactRootValue, shardID)
	}
	return sanitizeRelativePath(pathValue)
}

func stripTaskrunsStagingPrefix(pathValue string, artifactRoot string, shardID string) string {
	normalized := strings.TrimSpace(pathValue)
	if normalized == "" {
		return normalized
	}
	if artifactRoot != "" && strings.HasPrefix(normalized, artifactRoot+"/") {
		return strings.TrimPrefix(normalized, artifactRoot+"/")
	}
	marker := "/staging/shards/"
	idx := strings.Index(normalized, marker)
	if idx < 0 {
		return normalized
	}
	tail := normalized[idx+len(marker):]
	if tail == "" {
		return normalized
	}
	shardSlugValue := slugutil.Slugify(shardID)
	for _, candidate := range []string{strings.TrimSpace(shardID), shardSlugValue, path.Base(artifactRoot)} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && strings.HasPrefix(tail, candidate+"/") {
			return strings.TrimPrefix(tail, candidate+"/")
		}
	}
	parts := strings.SplitN(tail, "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return tail
}

func normalizeCanonicalPath(value string, task acpruntime.Task, document contracts.AuthoredDocument) string {
	candidate := filepath.ToSlash(strings.TrimSpace(value))
	if candidate != "" && isAllowedCanonicalRuntimeDocumentPath(candidate) && !strings.HasPrefix(candidate, "reports/taskruns/") {
		return candidate
	}
	return fallbackCanonicalPath(task, document.Path, document.ID)
}

func fallbackCanonicalPath(task acpruntime.Task, relPath string, docID string) string {
	normalized := sanitizeRelativePath(relPath)
	if normalized == "" {
		idSlug := slugutil.Slugify(docID)
		if idSlug == "" {
			idSlug = "artifact"
		}
		normalized = idSlug + ".md"
	}
	return path.Join("reports", "agent-outputs", "shards", shardSlug(task), normalized)
}

func fallbackDocumentPath(idx int) string {
	return fmt.Sprintf("artifact-%d.md", idx+1)
}

func fallbackEvidencePath(task acpruntime.Task) string {
	for _, candidate := range task.PathScopes {
		value := sanitizeRelativePath(candidate)
		if value == "" || value == "." {
			continue
		}
		return value
	}
	return "README.md"
}

func primaryRepoScope(task acpruntime.Task, manifestRepoScopes []string) string {
	for _, candidate := range append(append([]string{}, task.RepoScopes...), manifestRepoScopes...) {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	if trimmed := strings.TrimSpace(task.RepoScope); trimmed != "" {
		return trimmed
	}
	return "repo"
}

func shardSlug(task acpruntime.Task) string {
	for _, candidate := range []string{task.ShardID, task.DomainID, task.RepoScope} {
		if slug := slugutil.Slugify(candidate); slug != "" {
			return slug
		}
	}
	return "shard"
}

func sanitizeRelativePath(value string) string {
	candidate := strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(value), "./"))
	if candidate == "" {
		return ""
	}
	cleaned := path.Clean(candidate)
	for strings.HasPrefix(cleaned, "../") {
		cleaned = strings.TrimPrefix(cleaned, "../")
	}
	if cleaned == "." || cleaned == "" || cleaned == ".." {
		return ""
	}
	return cleaned
}

func ensureUniqueDocID(base string, seen map[string]int) string {
	return ensureUniqueID(base, seen, "doc")
}

func ensureUniqueCitationID(base string, seen map[string]int) string {
	return ensureUniqueID(base, seen, "cite")
}

func ensureUniqueID(base string, seen map[string]int, fallbackPrefix string) string {
	value := strings.TrimSpace(base)
	if value == "" {
		value = fallbackPrefix
	}
	if count, ok := seen[value]; ok {
		count++
		seen[value] = count
		return fmt.Sprintf("%s.%d", value, count)
	}
	seen[value] = 1
	return value
}

func ensureUniqueClaimID(base string, seen map[string]int, task acpruntime.Task) string {
	value := strings.TrimSpace(base)
	if value == "" {
		value = fmt.Sprintf("claim.%s", shardSlug(task))
	}
	if count, ok := seen[value]; ok {
		count++
		seen[value] = count
		return fmt.Sprintf("%s.%d", value, count)
	}
	seen[value] = 1
	return value
}

func slugFromPath(value string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(value)), filepath.Ext(strings.TrimSpace(value)))
	return slugutil.Slugify(base)
}

func titleFromPath(value string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(value)), filepath.Ext(strings.TrimSpace(value)))
	if base == "" {
		return "Runtime Artifact"
	}
	parts := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(base))
	for idx, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	if len(parts) == 0 {
		return "Runtime Artifact"
	}
	return strings.Join(parts, " ")
}

func extractLegacyCitationIDs(value any) []string {
	items := asObjectSlice(value)
	ids := []string{}
	for _, item := range items {
		if citationID := looseString(item["id"]); citationID != "" {
			ids = append(ids, citationID)
		}
	}
	return normalizeOrderedUniqueStrings(ids)
}

func asObjectSlice(value any) []map[string]any {
	objects := []map[string]any{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				objects = append(objects, object)
			}
		}
	case []map[string]any:
		objects = append(objects, typed...)
	}
	return objects
}

func looseString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func looseStringSlice(value any) []string {
	items := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := looseString(item); text != "" {
				items = append(items, text)
			}
		}
	case []string:
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				items = append(items, text)
			}
		}
	}
	return normalizeOrderedUniqueStrings(items)
}

func intValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		number, err := typed.Int64()
		if err == nil {
			return int(number), true
		}
	}
	return 0, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeOrderedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func isAllowedCanonicalRuntimeDocumentPath(rawPath string) bool {
	canonicalPath := path.Clean(strings.TrimSpace(rawPath))
	if canonicalPath == "." || canonicalPath == "" {
		return false
	}
	allowedPrefixes := []string{
		"reports/as-is/",
		"reports/findings/",
		"reports/coverage/",
		"reports/agent-outputs/",
		"reports/diagrams/",
		"proposals/",
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(canonicalPath, prefix) {
			return true
		}
	}
	return false
}

func CompatibilitySnapshotFromTaskResult(result contracts.TaskResult) contracts.CompatibilitySnapshot {
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

func canonicalAgentRole(task acpruntime.Task) string {
	if role := strings.TrimSpace(task.AgentRole); role != "" {
		return role
	}
	switch {
	case strings.HasSuffix(task.StepID, "step1.collect"):
		return "shard-analyst"
	case strings.HasSuffix(task.StepID, "step3.findings"):
		return "validator"
	default:
		return "runtime"
	}
}
