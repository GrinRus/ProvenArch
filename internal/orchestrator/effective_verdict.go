package orchestrator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/artifactaudit"
	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const effectiveVerdictFile = "effective-verdict.json"

func runtimeEffectiveVerdictPath(runID string) string {
	return path.Join(runtimeValidatorArtifactRoot(runID), effectiveVerdictFile)
}

func buildEffectiveVerdict(
	runID string,
	candidate contracts.ValidatorVerdict,
	providerRaw []byte,
	deterministic []contracts.ValidatorIssue,
	audit artifactaudit.Report,
	now time.Time,
) (contracts.EffectiveVerdict, error) {
	technical := append([]contracts.ValidatorIssue(nil), deterministic...)
	for _, issue := range audit.Issues {
		severity := strings.ToLower(strings.TrimSpace(issue.Severity))
		if severity != "error" && severity != "warning" {
			severity = "error"
		}
		technical = append(technical, contracts.ValidatorIssue{
			Code: issue.Code, Severity: severity, Message: issue.Message, Path: firstAuditPath(issue),
		})
	}
	technical = sortUniqueValidatorIssues(technical)

	technicalError := false
	for _, issue := range technical {
		if issue.Severity == "error" {
			technicalError = true
			break
		}
	}
	advisory := make([]contracts.AdvisoryValidatorIssue, 0, len(candidate.Issues))
	technicalKeys := map[string]struct{}{}
	for _, issue := range technical {
		technicalKeys[effectiveIssueKey(issue)] = struct{}{}
	}
	for _, issue := range candidate.Issues {
		key := effectiveIssueKey(issue)
		if _, matched := technicalKeys[key]; matched {
			continue
		}
		advisory = append(advisory, contracts.AdvisoryValidatorIssue{
			Source: "provider", MatchKey: key, Code: strings.TrimSpace(issue.Code), Severity: "warning",
			Message: strings.TrimSpace(issue.Message), Path: strings.TrimSpace(issue.Path),
			DocumentID: strings.TrimSpace(issue.DocumentID), CitationID: strings.TrimSpace(issue.CitationID),
		})
	}
	sort.Slice(advisory, func(i, j int) bool {
		if advisory[i].MatchKey != advisory[j].MatchKey {
			return advisory[i].MatchKey < advisory[j].MatchKey
		}
		return advisory[i].Message < advisory[j].Message
	})

	effective := contracts.EffectiveVerdict{
		Version: 1, Kind: "effective", Authority: "orchestrator", RunID: strings.TrimSpace(runID),
		GeneratedAt: now.UTC().Format(time.RFC3339), ProviderVerdictPath: runtimeValidatorVerdictPath(runID),
		ProviderVerdictSHA256: sha256Hex(providerRaw), Verdict: "PASS", Summary: "deterministic validation and selected-run audit passed",
		CheckedPaths: effectiveCheckedPaths(runID, candidate.CheckedPaths), FixedPaths: effectiveRunPaths(runID, candidate.FixedPaths),
		Findings: append([]contracts.Finding(nil), candidate.Findings...), Questions: append([]contracts.Question(nil), candidate.Questions...),
		TechnicalIssues: technical, AdvisoryIssues: advisory,
		Audit: contracts.EffectiveAuditSummary{Status: string(audit.Status), ErrorCount: audit.Summary.Error, WarningCount: audit.Summary.Warning, IssueCodes: auditIssueCodes(audit)},
	}
	if technicalError {
		effective.Verdict = "FAIL"
		effective.Summary = "deterministic validation or selected-run audit reported technical errors"
	}
	return effective, nil
}

func persistEffectiveVerdict(e *pipelineExecution, candidate contracts.ValidatorVerdict, providerRaw []byte, deterministic []contracts.ValidatorIssue, audit artifactaudit.Report) (contracts.EffectiveVerdict, error) {
	runID := strings.TrimSpace(e.runID)
	if runID == "" {
		runID = strings.TrimSpace(candidate.RunID)
	}
	if runID == "" && e.finalRunIndex != nil {
		runID = strings.TrimSpace(e.finalRunIndex.RunID)
	}
	now := time.Now().UTC()
	if e.clock != nil {
		now = e.clock().UTC()
	}
	effective, err := buildEffectiveVerdict(runID, candidate, providerRaw, deterministic, audit, now)
	if err != nil {
		return contracts.EffectiveVerdict{}, err
	}
	raw, err := json.MarshalIndent(effective, "", "  ")
	if err != nil {
		return contracts.EffectiveVerdict{}, fmt.Errorf("marshal effective verdict: %w", err)
	}
	raw = append(raw, '\n')
	parsed, err := contracts.ParseEffectiveVerdict(raw)
	if err != nil {
		return contracts.EffectiveVerdict{}, fmt.Errorf("parse effective verdict: %w", err)
	}
	if err := e.workspace.WriteFile(runtimeEffectiveVerdictPath(runID), raw); err != nil {
		return contracts.EffectiveVerdict{}, fmt.Errorf("write effective verdict: %w", err)
	}
	e.effectiveVerdict = &parsed
	e.addArtifacts(Artifact{Path: runtimeEffectiveVerdictPath(runID), Kind: "taskrun", Label: "Effective Technical Verdict"})
	e.recordConformanceDiagnostic(map[string]any{
		"effective_verdict_source": "orchestrator",
	})
	return parsed, nil
}

func effectiveIssueKey(issue contracts.ValidatorIssue) string {
	return strings.Join([]string{strings.TrimSpace(issue.Code), cleanEffectivePath(issue.Path), strings.TrimSpace(issue.DocumentID), strings.TrimSpace(issue.CitationID)}, "\x00")
}

func sortUniqueValidatorIssues(issues []contracts.ValidatorIssue) []contracts.ValidatorIssue {
	seen := map[string]contracts.ValidatorIssue{}
	for _, issue := range issues {
		key := effectiveIssueKey(issue) + "\x00" + strings.TrimSpace(issue.Severity) + "\x00" + strings.TrimSpace(issue.Message)
		seen[key] = issue
	}
	result := make([]contracts.ValidatorIssue, 0, len(seen))
	for _, issue := range seen {
		result = append(result, issue)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := effectiveIssueKey(result[i]), effectiveIssueKey(result[j])
		if left != right {
			return left < right
		}
		if result[i].Severity != result[j].Severity {
			return result[i].Severity < result[j].Severity
		}
		return result[i].Message < result[j].Message
	})
	return result
}

func effectiveCheckedPaths(runID string, candidate []string) []string {
	prefix := path.Join("reports", "taskruns", strings.TrimSpace(runID)) + "/"
	paths := []string{runtimeFinalRunIndexPath(runID), runtimeCitationIndexPath(runID)}
	for _, raw := range candidate {
		clean := cleanEffectivePath(raw)
		if clean != "" && strings.HasPrefix(clean, prefix) {
			paths = append(paths, clean)
		}
	}
	return sortedUniquePaths(paths)
}

func effectiveRunPaths(runID string, candidate []string) []string {
	prefix := path.Join("reports", "taskruns", strings.TrimSpace(runID)) + "/"
	paths := []string{}
	for _, raw := range candidate {
		clean := cleanEffectivePath(raw)
		if clean != "" && strings.HasPrefix(clean, prefix) {
			paths = append(paths, clean)
		}
	}
	return sortedUniquePaths(paths)
}

func sortedUniquePaths(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cleanEffectivePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") {
		return ""
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	return clean
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func firstAuditPath(issue artifactaudit.Issue) string {
	if strings.TrimSpace(issue.Path) != "" {
		return issue.Path
	}
	if len(issue.RelatedPaths) > 0 {
		return issue.RelatedPaths[0]
	}
	return ""
}

func auditIssueCodes(report artifactaudit.Report) []string {
	codes := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		if strings.TrimSpace(issue.Code) != "" {
			codes = append(codes, issue.Code)
		}
	}
	return sortedUniquePaths(codes)
}
