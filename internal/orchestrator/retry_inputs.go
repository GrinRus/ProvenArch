package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func copyRetryStaging(ws workspace.Root, parentRunID string, childRunID string, resumeStep string, requestedScopes []string) error {
	parentRunID = strings.TrimSpace(parentRunID)
	childRunID = strings.TrimSpace(childRunID)
	if parentRunID == "" || childRunID == "" || parentRunID == childRunID {
		return fmt.Errorf("retry parent and child run ids must be distinct and non-empty")
	}
	sourceRel := filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "staging"))
	sourceAbs, err := ws.Resolve(sourceRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent staging is unavailable")
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("parent staging is not a directory")
	}
	validatedShards, err := validatedReusableShardRoots(sourceAbs, parentRunID, resumeStep, requestedScopes)
	if err != nil {
		return err
	}
	if err := validateReusableFinalInputs(sourceAbs, parentRunID, resumeStep); err != nil {
		return err
	}
	if err := filepath.WalkDir(sourceAbs, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("retry staging contains unsupported symlink %q", entry.Name())
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourceAbs, current)
		if err != nil {
			return err
		}
		canonicalRel := filepath.ToSlash(rel)
		if strings.HasPrefix(canonicalRel, "shards/") {
			parts := strings.Split(canonicalRel, "/")
			if len(parts) < 3 || !validatedShards[parts[1]] {
				return nil
			}
		} else if !retryStagingPathReusable(canonicalRel, resumeStep, requestedScopes) {
			return nil
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		content, err = rebindRetryInputContent(content, canonicalRel, parentRunID, childRunID)
		if err != nil {
			return err
		}
		targetRel := filepath.ToSlash(filepath.Join("reports", "taskruns", childRunID, "staging", rel))
		if err := ws.WriteFile(targetRel, content); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err := normalizeRetryAggregateInputs(ws, childRunID, resumeStep); err != nil {
		return err
	}
	return copyReusableValidatorInput(ws, parentRunID, childRunID, resumeStep)
}

// ValidateRetryStaging verifies every parent shard that the retry intends to reuse.
func ValidateRetryStaging(ws workspace.Root, parentRunID, resumeStep string, requestedScopes []string) error {
	sourceRel := filepath.ToSlash(filepath.Join("reports", "taskruns", strings.TrimSpace(parentRunID), "staging"))
	sourceAbs, err := ws.Resolve(sourceRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(sourceAbs)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("parent staging is unavailable")
	}
	_, err = validatedReusableShardRoots(sourceAbs, strings.TrimSpace(parentRunID), resumeStep, requestedScopes)
	if err != nil {
		return err
	}
	if err := validateReusableFinalInputs(sourceAbs, strings.TrimSpace(parentRunID), resumeStep); err != nil {
		return err
	}
	return validateReusableValidatorInput(ws, strings.TrimSpace(parentRunID), resumeStep)
}

func rebindRetryInputContent(content []byte, relPath, parentRunID, childRunID string) ([]byte, error) {
	switch {
	case strings.HasPrefix(relPath, "shards/") && strings.HasSuffix(relPath, "/shard-pack-manifest.json"):
		manifest, err := contracts.ParseShardPackManifest(content)
		if err != nil {
			return nil, err
		}
		manifest.RunID = childRunID
		manifest.StepID = strings.Replace(manifest.StepID, parentRunID, childRunID, 1)
		parts := strings.Split(relPath, "/")
		manifest.ArtifactRoot = filepath.ToSlash(filepath.Join("reports", "taskruns", childRunID, "staging", "shards", parts[1]))
		return marshalRetryJSON(manifest)
	case relPath == "final/final-run-index.json":
		index, err := contracts.ParseFinalRunIndex(content)
		if err != nil {
			return nil, err
		}
		index.RunID = childRunID
		index.CitationIndexPath = replaceRetryRunPath(index.CitationIndexPath, parentRunID, childRunID)
		for i := range index.CanonicalDocuments {
			index.CanonicalDocuments[i].StagedPath = replaceRetryRunPath(index.CanonicalDocuments[i].StagedPath, parentRunID, childRunID)
		}
		return marshalRetryJSON(index)
	case relPath == "final/citation-index.json":
		index, err := contracts.ParseCitationIndex(content)
		if err != nil {
			return nil, err
		}
		index.RunID = childRunID
		return marshalRetryJSON(index)
	default:
		return content, nil
	}
}

func copyReusableValidatorInput(ws workspace.Root, parentRunID, childRunID, resumeStep string) error {
	if !strings.Contains(strings.ToLower(resumeStep), "proposals") {
		return nil
	}
	raw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "validator", "validator-verdict.json")))
	if err != nil {
		return err
	}
	verdict, err := contracts.ParseValidatorVerdict(raw)
	if err != nil {
		return err
	}
	verdict.RunID = childRunID
	for i := range verdict.CheckedPaths {
		verdict.CheckedPaths[i] = replaceRetryRunPath(verdict.CheckedPaths[i], parentRunID, childRunID)
	}
	for i := range verdict.FixedPaths {
		verdict.FixedPaths[i] = replaceRetryRunPath(verdict.FixedPaths[i], parentRunID, childRunID)
	}
	raw, err = marshalRetryJSON(verdict)
	if err != nil {
		return err
	}
	return ws.WriteFile(filepath.ToSlash(filepath.Join("reports", "taskruns", childRunID, "validator", "validator-verdict.json")), raw)
}

func validateReusableValidatorInput(ws workspace.Root, parentRunID, resumeStep string) error {
	if !strings.Contains(strings.ToLower(resumeStep), "proposals") {
		return nil
	}
	raw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "validator", "validator-verdict.json")))
	if err != nil {
		return fmt.Errorf("parent validator retry input is unavailable: %w", err)
	}
	verdict, err := contracts.ParseValidatorVerdict(raw)
	if err != nil {
		return fmt.Errorf("parent validator retry input is invalid: %w", err)
	}
	if strings.TrimSpace(verdict.RunID) != parentRunID || verdict.Verdict != "PASS" {
		return fmt.Errorf("parent validator retry input is not a matching PASS verdict")
	}
	return nil
}

func replaceRetryRunPath(value, parentRunID, childRunID string) string {
	return strings.Replace(value, "/"+parentRunID+"/", "/"+childRunID+"/", 1)
}

func marshalRetryJSON(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func normalizeRetryAggregateInputs(ws workspace.Root, childRunID, resumeStep string) error {
	resume := strings.ToLower(strings.TrimSpace(resumeStep))
	if !strings.Contains(resume, "findings") && !strings.Contains(resume, "proposals") {
		return nil
	}
	finalRaw, err := ws.ReadFile(runtimeFinalRunIndexPath(childRunID))
	if err != nil {
		return err
	}
	finalIndex, err := contracts.ParseFinalRunIndex(finalRaw)
	if err != nil {
		return err
	}
	keptDocs := map[string]struct{}{}
	documents := make([]contracts.FinalRunDocument, 0, len(finalIndex.CanonicalDocuments))
	for _, document := range finalIndex.CanonicalDocuments {
		if !retryStagingPathReusable("final/"+strings.TrimPrefix(document.CanonicalPath, "/"), resumeStep, nil) {
			continue
		}
		keptDocs[document.ID] = struct{}{}
		documents = append(documents, document)
	}
	finalIndex.CanonicalDocuments = documents
	for i := range finalIndex.Topics {
		ids := finalIndex.Topics[i].DocumentIDs[:0]
		for _, id := range finalIndex.Topics[i].DocumentIDs {
			if _, ok := keptDocs[id]; ok {
				ids = append(ids, id)
			}
		}
		finalIndex.Topics[i].DocumentIDs = ids
	}
	topics := finalIndex.Topics[:0]
	for _, topic := range finalIndex.Topics {
		if len(topic.DocumentIDs) > 0 {
			topics = append(topics, topic)
		}
	}
	finalIndex.Topics = topics
	citationRaw, err := ws.ReadFile(runtimeCitationIndexPath(childRunID))
	if err != nil {
		return err
	}
	citationIndex, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil {
		return err
	}
	keptCitations := map[string]struct{}{}
	citations := citationIndex.Citations[:0]
	for _, citation := range citationIndex.Citations {
		ids := citation.DocumentIDs[:0]
		for _, id := range citation.DocumentIDs {
			if _, ok := keptDocs[id]; ok {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			continue
		}
		citation.DocumentIDs = ids
		keptCitations[citation.ID] = struct{}{}
		citations = append(citations, citation)
	}
	citationIndex.Citations = citations
	for i := range finalIndex.CanonicalDocuments {
		ids := finalIndex.CanonicalDocuments[i].CitationIDs[:0]
		for _, id := range finalIndex.CanonicalDocuments[i].CitationIDs {
			if _, ok := keptCitations[id]; ok {
				ids = append(ids, id)
			}
		}
		finalIndex.CanonicalDocuments[i].CitationIDs = ids
	}
	finalRaw, err = marshalRetryJSON(finalIndex)
	if err != nil {
		return err
	}
	citationRaw, err = marshalRetryJSON(citationIndex)
	if err != nil {
		return err
	}
	if err := ws.WriteFile(runtimeFinalRunIndexPath(childRunID), finalRaw); err != nil {
		return err
	}
	return ws.WriteFile(runtimeCitationIndexPath(childRunID), citationRaw)
}

func (e *pipelineExecution) hydrateRetryInputs() error {
	if strings.TrimSpace(e.resumeFromStep) == "" {
		return nil
	}
	shardsRoot, err := e.workspace.Resolve(filepath.ToSlash(filepath.Join("reports", "taskruns", e.runID, "staging", "shards")))
	if err != nil {
		return err
	}
	entries, readErr := os.ReadDir(shardsRoot)
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, _, loadErr := loadShardPackManifestFromRoot(filepath.Join(shardsRoot, entry.Name()))
		if loadErr != nil {
			return fmt.Errorf("load retry shard %q: %w", entry.Name(), loadErr)
		}
		if manifest.RunID != e.runID {
			return fmt.Errorf("retry shard %q was not rebound to child run", entry.Name())
		}
		e.shardPacks = append(e.shardPacks, manifest)
	}
	sort.Slice(e.shardPacks, func(i, j int) bool { return e.shardPacks[i].ShardID < e.shardPacks[j].ShardID })
	resume := strings.ToLower(e.resumeFromStep)
	if strings.Contains(resume, "findings") || strings.Contains(resume, "proposals") {
		finalRaw, readErr := e.workspace.ReadFile(runtimeFinalRunIndexPath(e.runID))
		if readErr != nil {
			return readErr
		}
		finalIndex, parseErr := contracts.ParseFinalRunIndex(finalRaw)
		if parseErr != nil {
			return parseErr
		}
		citationRaw, readErr := e.workspace.ReadFile(runtimeCitationIndexPath(e.runID))
		if readErr != nil {
			return readErr
		}
		citationIndex, parseErr := contracts.ParseCitationIndex(citationRaw)
		if parseErr != nil {
			return parseErr
		}
		if finalIndex.RunID != e.runID || citationIndex.RunID != e.runID {
			return fmt.Errorf("retry aggregate inputs were not rebound to child run")
		}
		e.finalRunIndex, e.citationIndex = &finalIndex, &citationIndex
		e.findings = append([]contracts.Finding(nil), finalIndex.Semantic.Findings...)
		e.questions = append([]contracts.Question(nil), finalIndex.Semantic.Questions...)
		e.coverage = mergeCoverage(nil, &finalIndex.Semantic.Coverage)
	}
	if strings.Contains(resume, "proposals") {
		verdictRaw, readErr := e.workspace.ReadFile(runtimeValidatorVerdictPath(e.runID))
		if readErr != nil {
			return readErr
		}
		verdict, parseErr := contracts.ParseValidatorVerdict(verdictRaw)
		if parseErr != nil {
			return parseErr
		}
		if verdict.RunID != e.runID || verdict.Verdict != "PASS" {
			return fmt.Errorf("retry validator input was not rebound to a matching PASS verdict")
		}
		e.validatorVerdict = &verdict
	}
	return nil
}

func validateReusableFinalInputs(stagingRoot, parentRunID, resumeStep string) error {
	resumeStep = strings.ToLower(strings.TrimSpace(resumeStep))
	if !strings.Contains(resumeStep, "findings") && !strings.Contains(resumeStep, "proposals") {
		return nil
	}
	finalRoot := filepath.Join(stagingRoot, "final")
	indexRaw, err := os.ReadFile(filepath.Join(finalRoot, "final-run-index.json"))
	if err != nil {
		return fmt.Errorf("parent aggregated retry input is unavailable: %w", err)
	}
	index, err := contracts.ParseFinalRunIndex(indexRaw)
	if err != nil {
		return fmt.Errorf("parent aggregated retry input is invalid: %w", err)
	}
	if strings.TrimSpace(index.RunID) != parentRunID {
		return fmt.Errorf("parent aggregated retry input belongs to run %q", index.RunID)
	}
	citationRaw, err := os.ReadFile(filepath.Join(finalRoot, "citation-index.json"))
	if err != nil {
		return fmt.Errorf("parent citation retry input is unavailable: %w", err)
	}
	citations, err := contracts.ParseCitationIndex(citationRaw)
	if err != nil || strings.TrimSpace(citations.RunID) != parentRunID {
		if err != nil {
			return fmt.Errorf("parent citation retry input is invalid: %w", err)
		}
		return fmt.Errorf("parent citation retry input belongs to run %q", citations.RunID)
	}
	if err := validateRetryAggregateBindings(index, citations); err != nil {
		return fmt.Errorf("parent aggregate retry input is inconsistent: %w", err)
	}
	for _, document := range index.CanonicalDocuments {
		if !retryStagingPathReusable("final/"+strings.TrimPrefix(document.CanonicalPath, "/"), resumeStep, nil) {
			continue
		}
		prefix := filepath.ToSlash(filepath.Join("reports", "taskruns", parentRunID, "staging", "final")) + "/"
		stagedPath := filepath.ToSlash(filepath.Clean(document.StagedPath))
		if !strings.HasPrefix(stagedPath, prefix) {
			return fmt.Errorf("parent retry document %q is outside the parent final staging", document.ID)
		}
		candidate := filepath.Join(finalRoot, filepath.FromSlash(strings.TrimPrefix(stagedPath, prefix)))
		rel, relErr := filepath.Rel(finalRoot, candidate)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("parent retry document %q escapes final staging", document.ID)
		}
		info, statErr := os.Stat(candidate)
		if statErr != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("parent retry document %q is unavailable", document.ID)
		}
	}
	return nil
}

func validateRetryAggregateBindings(index contracts.FinalRunIndex, citations contracts.CitationIndex) error {
	documentsByID := map[string]contracts.FinalRunDocument{}
	for _, document := range index.CanonicalDocuments {
		documentsByID[document.ID] = document
	}
	citationsByID := map[string]contracts.DocumentCitation{}
	for _, citation := range citations.Citations {
		citationsByID[citation.ID] = citation
	}
	for _, document := range index.CanonicalDocuments {
		for _, citationID := range document.CitationIDs {
			citation, ok := citationsByID[citationID]
			if !ok || !containsTrimmedString(citation.DocumentIDs, document.ID) {
				return fmt.Errorf("document %q and citation %q are not reciprocal", document.ID, citationID)
			}
		}
	}
	for _, citation := range citations.Citations {
		for _, documentID := range citation.DocumentIDs {
			document, ok := documentsByID[documentID]
			if !ok || !containsTrimmedString(document.CitationIDs, citation.ID) {
				return fmt.Errorf("citation %q and document %q are not reciprocal", citation.ID, documentID)
			}
		}
	}
	return nil
}

func validatedReusableShardRoots(stagingRoot, parentRunID, resumeStep string, requestedScopes []string) (map[string]bool, error) {
	result := map[string]bool{}
	if strings.Contains(strings.ToLower(resumeStep), "constitution") || (strings.Contains(strings.ToLower(resumeStep), "collect") && len(requestedScopes) == 0) {
		return result, nil
	}
	shardsRoot := filepath.Join(stagingRoot, "shards")
	entries, err := os.ReadDir(shardsRoot)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.Contains(strings.ToLower(resumeStep), "collect") && retryScopeMatchesValues(requestedScopes, []string{entry.Name()}) {
			continue
		}
		root := filepath.Join(shardsRoot, entry.Name())
		if err := artifactquality.ValidateCollectManifestInRoot(root); err != nil {
			return nil, fmt.Errorf("parent retry input shard %q is not reusable: %w", entry.Name(), err)
		}
		raw, err := os.ReadFile(filepath.Join(root, "shard-pack-manifest.json"))
		if err != nil {
			return nil, err
		}
		manifest, err := contracts.ParseShardPackManifest(raw)
		if err != nil {
			return nil, err
		}
		expectedRelativeRoot := filepath.Clean(filepath.Join("reports", "taskruns", parentRunID, "staging", "shards", entry.Name()))
		artifactRoot := filepath.Clean(manifest.ArtifactRoot)
		rootMatches := artifactRoot == filepath.Clean(root) || artifactRoot == expectedRelativeRoot
		if strings.TrimSpace(manifest.RunID) != parentRunID || !strings.HasSuffix(strings.TrimSpace(manifest.StepID), ".step1.collect") || strings.TrimSpace(manifest.ShardID) != entry.Name() || !rootMatches {
			return nil, fmt.Errorf("parent retry input shard %q has mismatched task identity (run=%q step=%q shard=%q artifact_root=%q expected_root=%q)", entry.Name(), manifest.RunID, manifest.StepID, manifest.ShardID, manifest.ArtifactRoot, root)
		}
		if strings.Contains(strings.ToLower(resumeStep), "collect") && retryScopeMatchesManifest(requestedScopes, manifest) {
			continue
		}
		result[entry.Name()] = true
	}
	return result, nil
}

func retryScopeMatchesManifest(scopes []string, manifest contracts.ShardPackManifest) bool {
	values := append([]string{manifest.ShardID, manifest.DomainID}, manifest.RepoScopes...)
	return retryScopeMatchesValues(scopes, values)
}

func retryScopeMatchesValues(scopes, values []string) bool {
	for _, scope := range scopes {
		scope = strings.ToLower(strings.TrimSpace(scope))
		for _, value := range values {
			if scope != "" && scope == strings.ToLower(strings.TrimSpace(value)) {
				return true
			}
		}
	}
	return false
}

func retryStagingPathReusable(relPath, resumeStep string, requestedScopes []string) bool {
	relPath = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(relPath)), "./")
	resumeStep = strings.ToLower(strings.TrimSpace(resumeStep))
	if relPath == "" || relPath == "." || strings.HasPrefix(relPath, "../") {
		return false
	}
	switch {
	case strings.Contains(resumeStep, "constitution"):
		return false
	case strings.Contains(resumeStep, "collect"):
		if len(requestedScopes) == 0 || !strings.HasPrefix(relPath, "shards/") {
			return false
		}
		parts := strings.Split(relPath, "/")
		if len(parts) > 1 && retryScopeMatchesValues(requestedScopes, []string{parts[1]}) {
			return false
		}
		return true
	case strings.Contains(resumeStep, "asis"):
		return strings.HasPrefix(relPath, "shards/")
	case strings.Contains(resumeStep, "findings"):
		return !strings.Contains(relPath, "reports/findings/") && !strings.Contains(relPath, "reports/coverage/") && !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	case strings.Contains(resumeStep, "proposals"):
		return !strings.Contains(relPath, "proposals/") && !strings.Contains(relPath, "reports/changelog/")
	default:
		return false
	}
}
