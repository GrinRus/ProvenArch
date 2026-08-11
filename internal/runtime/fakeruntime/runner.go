package fakeruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/qa"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

const providerFake acpruntime.Provider = "fake"

type Runner struct{}

func (Runner) RuntimeMeta() contracts.RuntimeMeta {
	return contracts.RuntimeMeta{Name: string(providerFake), Version: "fake"}
}

func (Runner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := runFakePermissionFixture(task); err != nil {
		return acpruntime.Result{}, err
	}
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderQA {
		if err := persistFakeQAAnswer(task); err != nil {
			return acpruntime.Result{}, err
		}
		if err := providercommon.ValidateRuntimeArtifacts(task, providerFake); err != nil {
			return acpruntime.Result{}, err
		}
		return acpruntime.Result{
			Execution: acpruntime.NewExecution(task, providerFake, "fake", "succeeded", task.StartedAtUTC.UTC().Add(2*time.Second), nil),
		}, nil
	}
	semantic := fakeSemanticSnapshot(task)
	summary := fakeSummary(task)
	var verdict *contracts.ValidatorVerdict
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderStep3Findings {
		verdict = &contracts.ValidatorVerdict{
			Version:      1,
			RunID:        task.RunID,
			GeneratedAt:  task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
			Verdict:      "PASS",
			Summary:      "Fake validator verdict accepted",
			CheckedPaths: collectValidatorCheckedPaths(task),
			Findings:     append([]contracts.Finding(nil), semantic.Findings...),
			Questions:    append([]contracts.Question(nil), semantic.Questions...),
		}
	}
	if err := persistRuntimeArtifacts(task, summary, semantic, verdict); err != nil {
		return acpruntime.Result{}, err
	}
	if err := providercommon.ValidateRuntimeArtifacts(task, providerFake); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, providerFake, "fake", "succeeded", task.StartedAtUTC.UTC().Add(2*time.Second), nil),
	}, nil
}

func persistFakeQAAnswer(task acpruntime.Task) error {
	if err := os.MkdirAll(filepath.Clean(task.WriteRoot), 0o755); err != nil {
		return err
	}
	contextPack, _ := loadQAContextPack(task.ContextPackPath)
	citations := []qa.Citation{}
	for _, doc := range contextPack.Documents {
		if strings.TrimSpace(doc.Path) == "" {
			continue
		}
		citations = append(citations, qa.Citation{
			Path:   doc.Path,
			Reason: "selected from QA context pack by fake runtime",
		})
		if len(citations) >= 3 {
			break
		}
	}
	if len(citations) == 0 {
		citations = []qa.Citation{}
	}
	answer := qa.Answer{
		Version:     1,
		RunID:       strings.TrimSpace(task.RunID),
		Question:    strings.TrimSpace(task.Question),
		Answer:      fmt.Sprintf("Fake runtime QA inspected %d workspace artifact(s) for this question.", len(contextPack.Documents)),
		Citations:   citations,
		Unresolved:  []string{},
		Confidence:  0.7,
		Provider:    string(providerFake),
		GeneratedAt: task.StartedAtUTC.UTC().Add(2 * time.Second).Format(time.RFC3339),
	}
	if len(contextPack.Documents) == 0 {
		answer.Unresolved = []string{"no indexed workspace evidence available in context pack"}
		answer.Confidence = 0.2
	}
	raw, err := json.MarshalIndent(answer, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(filepath.Join(filepath.Clean(task.WriteRoot), "qa-answer.json"), raw, 0o644)
}

func loadQAContextPack(path string) (qa.ContextPack, error) {
	if strings.TrimSpace(path) == "" {
		return qa.ContextPack{}, fmt.Errorf("context pack path is required")
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return qa.ContextPack{}, err
	}
	var pack qa.ContextPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		return qa.ContextPack{}, err
	}
	return pack, nil
}

func runFakePermissionFixture(task acpruntime.Task) error {
	if strings.TrimSpace(task.RuntimePermissions.Mode) != acpruntime.PermissionModeManaged || task.OnPermissionRequest == nil {
		return nil
	}
	if len(task.ReadContextRoots) > 0 {
		decision := task.OnPermissionRequest(acpruntime.PermissionRequest{
			RequestID:     "fake-read-context",
			RunID:         task.RunID,
			StepID:        task.StepID,
			Provider:      providerFake,
			Action:        "read",
			PathOrCommand: task.ReadContextRoots[0],
			Reason:        "fake permission fixture read",
		})
		if !decision.Approved() {
			return acpruntime.WrapRunnerError(providerFake, acpruntime.ErrorCodePermissionRequired, fmt.Sprintf("fake read permission was not approved: %s", decision.RuleID), nil)
		}
	}
	if strings.TrimSpace(task.WriteRoot) != "" {
		decision := task.OnPermissionRequest(acpruntime.PermissionRequest{
			RequestID:     "fake-write-root",
			RunID:         task.RunID,
			StepID:        task.StepID,
			Provider:      providerFake,
			Action:        "write",
			PathOrCommand: filepath.Join(task.WriteRoot, "permission-fixture.tmp"),
			Reason:        "fake permission fixture write",
		})
		if !decision.Approved() {
			return acpruntime.WrapRunnerError(providerFake, acpruntime.ErrorCodePermissionRequired, fmt.Sprintf("fake write permission was not approved: %s", decision.RuleID), nil)
		}
	}
	return nil
}

func fakeSummary(task acpruntime.Task) string {
	switch acpruntime.StepProviderKeyForStepID(task.StepID) {
	case acpruntime.StepProviderStep0Constitution:
		return "Fake constitution draft completed"
	case acpruntime.StepProviderStep1Collect:
		return "Fake collect shard completed"
	case acpruntime.StepProviderStep2AsIs:
		return "Fake as-is draft completed"
	case acpruntime.StepProviderStep3Findings:
		return "Fake validator findings completed"
	case acpruntime.StepProviderStep4Proposals:
		return "Fake proposals draft completed"
	default:
		return "Fake runtime completed"
	}
}

func fakeSemanticSnapshot(task acpruntime.Task) contracts.SemanticSnapshot {
	repoScopes := append([]string(nil), task.RepoScopes...)
	sort.Strings(repoScopes)
	primaryRepo := ""
	if len(repoScopes) > 0 {
		primaryRepo = repoScopes[0]
	}
	if primaryRepo == "" {
		primaryRepo = "stub-repo"
	}
	evidencePath := fallbackEvidencePath(task)
	serviceID := "svc." + slugutil.Slugify(primaryRepo)
	snapshot := contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{
			Observed: []string{"services", "entrypoints"},
			Missing:  []string{"owner mappings", "ci-cd evidence", "api contracts"},
			Notes:    []string{"fake artifact-only runner output"},
		},
		Questions: []contracts.Question{
			{
				ID:         "q.owner." + serviceID,
				Text:       fmt.Sprintf("Who owns %s?", serviceID),
				Priority:   "high",
				RelatedIDs: []string{serviceID},
			},
		},
		Entities: []contracts.Entity{
			{
				ID:   serviceID,
				Type: "service",
				Name: humanizeServiceName(primaryRepo),
				Attributes: map[string]any{
					"repo_scope": primaryRepo,
					"runtime":    "fake",
				},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence: []contracts.Evidence{
						{Repo: primaryRepo, Path: evidencePath},
					},
				},
			},
		},
		Edges: []contracts.Edge{},
		Findings: []contracts.Finding{
			{
				ID:          "finding.owner." + serviceID,
				Severity:    "medium",
				Title:       "Owner mapping is missing",
				Description: "Fake runner keeps an explicit ownership gap for deterministic validation coverage.",
				RuleID:      "rule.owner.required",
				RelatedIDs:  []string{serviceID},
				Provenance: contracts.Provenance{
					Kind:       "inference",
					Confidence: 0.6,
					Evidence: []contracts.Evidence{
						{Repo: primaryRepo, Path: evidencePath},
					},
				},
			},
		},
	}
	if acpruntime.StepProviderKeyForStepID(task.StepID) == acpruntime.StepProviderStep3Findings {
		snapshot.Coverage.Observed = []string{"validator sweep", "staged findings"}
		snapshot.Coverage.Missing = []string{"manual rollout review"}
	}
	return snapshot
}

func humanizeServiceName(repo string) string {
	if strings.TrimSpace(repo) == "" {
		return "Stub Service"
	}
	parts := strings.FieldsFunc(repo, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
