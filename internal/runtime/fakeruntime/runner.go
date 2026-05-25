package fakeruntime

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
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
			CheckedPaths: []string{"reports/taskruns/" + task.RunID + "/staging/final"},
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
						{Repo: primaryRepo, Path: "README.md"},
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
						{Repo: primaryRepo, Path: "README.md"},
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
