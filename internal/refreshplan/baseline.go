package refreshplan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func LoadLatestValidBaseline(ws workspace.Root, candidateRunIDs []string) (*SourceRevisions, PriorEvidence) {
	for _, runID := range candidateRunIDs {
		runID = strings.TrimSpace(runID)
		if runID == "" {
			continue
		}
		finalRaw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "final-run-index.json")))
		if err != nil {
			continue
		}
		index, err := contracts.ParseFinalRunIndex(finalRaw)
		if err != nil || index.RunID != runID {
			continue
		}
		verdictRaw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "validator", "validator-verdict.json")))
		if err != nil {
			continue
		}
		verdict, err := contracts.ParseValidatorVerdict(verdictRaw)
		if err != nil || verdict.RunID != runID || strings.ToUpper(verdict.Verdict) != "PASS" {
			continue
		}
		revisionRaw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "source-revisions.json")))
		if err != nil {
			continue
		}
		revisions, err := ParseSourceRevisions(revisionRaw)
		if err != nil || revisions.RunID != runID {
			continue
		}
		return &revisions, loadPriorEvidence(ws, runID, index)
	}
	return nil, PriorEvidence{Readable: false, ArtifactShards: map[string][]string{}, CitationDocuments: map[string][]string{}, DocumentPaths: map[string]string{}, ProvenanceDomains: map[string][]string{}, ProvenanceArtifacts: map[string][]string{}, AllCanonicalPaths: []string{}}
}

func loadPriorEvidence(ws workspace.Root, runID string, index contracts.FinalRunIndex) PriorEvidence {
	evidence := PriorEvidence{Readable: true, ArtifactShards: map[string][]string{}, CitationDocuments: map[string][]string{}, DocumentPaths: map[string]string{}, ProvenanceDomains: map[string][]string{}, ProvenanceArtifacts: map[string][]string{}, AllCanonicalPaths: []string{}, Shards: []ShardScope{}}
	for _, doc := range index.CanonicalDocuments {
		evidence.ArtifactShards[doc.CanonicalPath] = uniqueSorted(doc.SourceShards...)
		evidence.DocumentPaths[doc.ID] = doc.CanonicalPath
		evidence.AllCanonicalPaths = append(evidence.AllCanonicalPaths, doc.CanonicalPath)
	}
	for _, entity := range index.Semantic.Entities {
		artifact := "model/entities/" + strings.ReplaceAll(entity.ID, "/", "_") + ".yaml"
		evidence.AllCanonicalPaths = append(evidence.AllCanonicalPaths, artifact)
		for _, source := range entity.Provenance.Evidence {
			key := source.Repo + "\x00" + filepath.ToSlash(filepath.Clean(source.Path))
			evidence.ProvenanceArtifacts[key] = uniqueSorted(append(evidence.ProvenanceArtifacts[key], artifact)...)
			if strings.EqualFold(entity.Type, "domain") || strings.HasPrefix(entity.ID, "domain.") {
				evidence.ProvenanceDomains[key] = uniqueSorted(append(evidence.ProvenanceDomains[key], entity.ID)...)
			}
		}
	}
	for _, edge := range index.Semantic.Edges {
		artifact := "model/edges/" + strings.ReplaceAll(edge.ID, "/", "_") + ".yaml"
		evidence.AllCanonicalPaths = append(evidence.AllCanonicalPaths, artifact)
		for _, source := range edge.Provenance.Evidence {
			key := source.Repo + "\x00" + filepath.ToSlash(filepath.Clean(source.Path))
			evidence.ProvenanceArtifacts[key] = uniqueSorted(append(evidence.ProvenanceArtifacts[key], artifact)...)
		}
	}
	citationRaw, err := ws.ReadFile(filepath.ToSlash(filepath.Join("reports", "taskruns", runID, "staging", "final", "citation-index.json")))
	if err != nil {
		evidence.Readable = false
	} else if citations, parseErr := contracts.ParseCitationIndex(citationRaw); parseErr != nil || citations.RunID != runID {
		evidence.Readable = false
	} else {
		for _, citation := range citations.Citations {
			key := citation.Repo + "\x00" + filepath.ToSlash(filepath.Clean(citation.Path))
			evidence.CitationDocuments[key] = uniqueSorted(append(evidence.CitationDocuments[key], citation.DocumentIDs...)...)
		}
	}
	absRoot, err := ws.Resolve("reports/taskruns")
	if err != nil {
		evidence.Readable = false
		return evidence
	}
	matches, err := filepath.Glob(filepath.Join(absRoot, runID+"-*-step1-collect-shard-plan*.json"))
	if err != nil {
		evidence.Readable = false
		return evidence
	}
	type planItem struct {
		ShardID    string   `json:"shard_id"`
		RepoScopes []string `json:"repo_scopes"`
		PathScopes []string `json:"path_scopes"`
	}
	type planFile struct {
		DomainID string     `json:"domain_id"`
		Items    []planItem `json:"items"`
	}
	for _, filename := range matches {
		raw, readErr := os.ReadFile(filename)
		if readErr != nil {
			evidence.Readable = false
			continue
		}
		var plan planFile
		if json.Unmarshal(raw, &plan) != nil {
			evidence.Readable = false
			continue
		}
		for _, item := range plan.Items {
			evidence.Shards = append(evidence.Shards, ShardScope{ShardID: item.ShardID, DomainID: plan.DomainID, RepoScopes: uniqueSorted(item.RepoScopes...), PathScopes: uniqueSorted(item.PathScopes...)})
		}
	}
	if len(matches) == 0 {
		evidence.Readable = false
	}
	sort.Strings(evidence.AllCanonicalPaths)
	sort.Slice(evidence.Shards, func(i, j int) bool { return evidence.Shards[i].ShardID < evidence.Shards[j].ShardID })
	return evidence
}
