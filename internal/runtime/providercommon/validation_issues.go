package providercommon

import (
	"errors"
	"os"
	"sort"
	"strings"
)

type validationIssueClass string

const (
	validationIssueMissing    validationIssueClass = "missing"
	validationIssueStructural validationIssueClass = "structural"
	validationIssueSemantic   validationIssueClass = "semantic"
	validationIssueWriteSet   validationIssueClass = "write_set"
)

type validationIssue struct {
	Code  string
	Class validationIssueClass
	Path  string
}

type validationIssueSet struct {
	Items []validationIssue
	index map[string]struct{}
}

type recoveryTransition struct {
	IssueCode   string
	TargetStage string
	MaxAttempts int
}

const (
	issueMissingArtifact          = "artifact.missing"
	issueCollectRepoEvidence      = "collect.repo_evidence_missing"
	issueCollectProcessContent    = "collect.process_contaminated"
	issueCollectWriteSet          = "collect.write_set"
	issueCollectAuthoredMarkdown  = "collect.authored_markdown"
	issueCollectBootstrap         = "collect.bootstrap_only"
	issueCollectDuplicateCitation = "collect.citation_id_duplicate"
	issueCollectQuestionText      = "collect.question_text"
	issueCollectClaimBinding      = "collect.claim_binding"
	issueCollectDocumentBinding   = "collect.document_binding"
	issueCollectCitationReference = "collect.citation_reference"
	issueCollectTaskIdentity      = "collect.task_identity"
	issueCollectSemanticScaffold  = "collect.semantic_scaffold"
	issueCollectEmptyPayload      = "collect.empty_payload"
	issueDraftManifestParse       = "draft.manifest_parse"
	issueDraftManifestOutputs     = "draft.manifest_outputs"
	issueDraftUnknownField        = "draft.unknown_field"
	issueDraftMalformedMarkdown   = "draft.malformed_markdown"
	issueDraftBootstrap           = "draft.bootstrap_only"
	issueDraftNoopScaffold        = "draft.enrichment_noop_scaffold"
	issueDraftDownstreamIndex     = "draft.downstream_index_claim"
	issueDraftShardStatus         = "draft.shard_status"
	issueDraftArchitectureHome    = "draft.architecture_home_contamination"
	issueDraftMarkerCleanup       = "draft.marker_cleanup"
	issueDraftFindingLinkage      = "draft.finding_linkage"
	issueDraftProposalSection     = "draft.proposal_section"
	issueDraftEvidence            = "draft.evidence"
	issueDraftOperatorSummary     = "draft.operator_summary"
	issueDraftEmptyShardEvidence  = "draft.empty_shard_evidence"
	issueDraftWriteRootForbidden  = "draft.write_root_forbidden"
	issueDraftFinalRootForbidden  = "draft.final_root_forbidden"
	issueRuntimeDraftManifest     = "artifact.runtime_draft_manifest"
	issueShardPackManifest        = "artifact.shard_pack_manifest"
	issueValidatorVerdict         = "artifact.validator_verdict"
	issueExpectedFile             = "artifact.expected_file"
)

func classifyValidationIssues(err error) validationIssueSet {
	set := validationIssueSet{Items: []validationIssue{}, index: map[string]struct{}{}}
	if err == nil {
		return set
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	add := func(code string, class validationIssueClass) {
		if _, exists := set.index[code]; exists {
			return
		}
		set.index[code] = struct{}{}
		set.Items = append(set.Items, validationIssue{Code: code, Class: class})
	}
	if errors.Is(err, os.ErrNotExist) || strings.Contains(text, "no such file or directory") ||
		(strings.Contains(text, "is unavailable") && !strings.Contains(text, "claims current-run")) ||
		(strings.Contains(text, "required") && strings.Contains(text, "missing")) {
		add(issueMissingArtifact, validationIssueMissing)
	}
	rules := []struct {
		code    string
		class   validationIssueClass
		markers []string
	}{
		{issueCollectRepoEvidence, validationIssueSemantic, []string{"repo evidence path"}},
		{issueCollectProcessContent, validationIssueSemantic, []string{"process-contaminated", "process contaminated"}},
		{issueCollectWriteSet, validationIssueWriteSet, []string{"outside shard-pack-manifest.json", "wrote forbidden files"}},
		{issueCollectAuthoredMarkdown, validationIssueSemantic, []string{"authored markdown", "empty authored markdown"}},
		{issueCollectBootstrap, validationIssueSemantic, []string{"bootstrap-only"}},
		{issueCollectSemanticScaffold, validationIssueSemantic, []string{"semantic snapshot is bootstrap-only collect scaffold"}},
		{issueCollectEmptyPayload, validationIssueStructural, []string{"shard-pack-manifest.schema.json validation failed: empty payload"}},
		{issueCollectTaskIdentity, validationIssueStructural, []string{"shard pack manifest task identity is invalid:"}},
		{issueDraftManifestParse, validationIssueStructural, []string{"parse runtime draft manifest:"}},
		{issueDraftManifestOutputs, validationIssueSemantic, []string{"runtime draft manifest outputs are invalid:"}},
		{issueDraftUnknownField, validationIssueStructural, []string{"unknown field"}},
		{issueDraftMalformedMarkdown, validationIssueSemantic, []string{"malformed markdown inline-code"}},
		{issueDraftBootstrap, validationIssueSemantic, []string{"bootstrap-only placeholder draft content"}},
		{issueDraftNoopScaffold, validationIssueSemantic, []string{"draft_artifact_enrichment_noop_or_scaffold"}},
		{issueDraftDownstreamIndex, validationIssueSemantic, []string{"claims current-run final/citation indexes are unavailable", "claims current-run final-run-index has zero observed documents"}},
		{issueDraftShardStatus, validationIssueSemantic, []string{"generic conditional shard-gap wording", "does not report exact current-run shard status", "does not report exact current-run proposal shard completeness", "does not report exact current-run shard completeness"}},
		{issueDraftArchitectureHome, validationIssueSemantic, []string{"architecture home contains runtime/process narration, manifest recap, or unsupported confidence language"}},
		{issueDraftMarkerCleanup, validationIssueSemantic, []string{"mentions downstream or runtime-only evidence in step0 constitution content", "proposal content references taskrun staging paths"}},
		{issueDraftFindingLinkage, validationIssueSemantic, []string{"does not reference any current-run finding id", "uses synthetic current-run finding placeholder", "claims no structured finding summary", "does not link medium/high current-run findings", "uses markdown table for medium/high actionable findings"}},
		{issueDraftProposalSection, validationIssueSemantic, []string{"is missing substantive proposal section", "is missing substantive proposal changelog section"}},
		{issueDraftEvidence, validationIssueSemantic, []string{"does not include concrete repo/path, citation, or staged artifact evidence references"}},
		{issueDraftOperatorSummary, validationIssueSemantic, []string{"does not include a decision-ready operator summary"}},
		{issueDraftEmptyShardEvidence, validationIssueSemantic, []string{"claims staging shard evidence is empty"}},
		{issueDraftWriteRootForbidden, validationIssueWriteSet, []string{"draft repair wrote forbidden write_root files"}},
		{issueDraftFinalRootForbidden, validationIssueWriteSet, []string{"draft repair wrote forbidden draft_final_root files"}},
		{issueRuntimeDraftManifest, validationIssueStructural, []string{"runtime draft manifest"}},
		{issueShardPackManifest, validationIssueStructural, []string{"shard pack manifest"}},
		{issueValidatorVerdict, validationIssueStructural, []string{"validator verdict"}},
		{issueExpectedFile, validationIssueStructural, []string{"must point to a file"}},
	}
	for _, rule := range rules {
		matched := false
		for _, marker := range rule.markers {
			if strings.Contains(text, marker) {
				matched = true
				break
			}
		}
		if matched {
			add(rule.code, rule.class)
		}
	}
	if strings.Contains(text, "citations") && strings.Contains(text, ".id must be unique") {
		add(issueCollectDuplicateCitation, validationIssueStructural)
	}
	if strings.Contains(text, "semantic/questions") && strings.Contains(text, "text") {
		add(issueCollectQuestionText, validationIssueStructural)
	}
	if strings.Contains(text, ".claim_ids is required") ||
		(strings.Contains(text, "/claim_ids") && containsAny(text, "minitems", "minimum 1 items required")) {
		add(issueCollectClaimBinding, validationIssueStructural)
	}
	if strings.Contains(text, ".document_ids is required") ||
		(strings.Contains(text, "/document_ids") && containsAny(text, "minitems", "minimum 1 items required")) {
		add(issueCollectDocumentBinding, validationIssueStructural)
	}
	if strings.Contains(text, "documents") && strings.Contains(text, "citation_ids") && strings.Contains(text, "references") {
		add(issueCollectCitationReference, validationIssueStructural)
	}
	for _, path := range missingRepoEvidencePathsFromText(err.Error()) {
		set.Items = append(set.Items, validationIssue{Code: issueCollectRepoEvidence, Class: validationIssueSemantic, Path: path})
	}
	sort.SliceStable(set.Items, func(i, j int) bool {
		if set.Items[i].Code == set.Items[j].Code {
			return set.Items[i].Path < set.Items[j].Path
		}
		return set.Items[i].Code < set.Items[j].Code
	})
	return set
}

func (s validationIssueSet) Has(code string) bool {
	_, ok := s.index[code]
	return ok
}

func (s validationIssueSet) HasAny(codes ...string) bool {
	for _, code := range codes {
		if s.Has(code) {
			return true
		}
	}
	return false
}

func (s validationIssueSet) Allows(currentStage string, transition recoveryTransition) bool {
	return transition.MaxAttempts > 0 &&
		strings.TrimSpace(currentStage) != transition.TargetStage &&
		s.Has(transition.IssueCode)
}

func containsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func missingRepoEvidencePathsFromText(text string) []string {
	const marker = `repo evidence path "`
	paths := []string{}
	seen := map[string]struct{}{}
	for {
		index := strings.Index(strings.ToLower(text), marker)
		if index < 0 {
			break
		}
		text = text[index+len(marker):]
		end := strings.Index(text, `"`)
		if end < 0 {
			break
		}
		value := strings.TrimSpace(text[:end])
		if value != "" {
			if _, exists := seen[value]; !exists {
				seen[value] = struct{}{}
				paths = append(paths, value)
			}
		}
		text = text[end+1:]
	}
	sort.Strings(paths)
	return paths
}
