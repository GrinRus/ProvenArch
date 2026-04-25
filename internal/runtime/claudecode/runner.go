package claudecode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
)

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CLAUDE_CMD"))
	}
	if command == "" {
		command = "claude-code"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	command := r.commandName()
	if _, err := exec.LookPath(command); err != nil {
		return acpruntime.WrapRunnerError(
			acpruntime.ProviderClaudeCode,
			acpruntime.ErrorCodeRunnerUnavailable,
			fmt.Sprintf("headless provider %q command %q is unavailable: %v", acpruntime.ProviderClaudeCode, command, err),
			err,
		)
	}
	return nil
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	return providercommon.RunHeadlessProvider(ctx, task, claudeAdapter{runner: r})
}

type claudeAdapter struct {
	runner HeadlessRunner
}

func (a claudeAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderClaudeCode
}

func (a claudeAdapter) RuntimeVersion() string {
	return "headless"
}

func (a claudeAdapter) CommandSpec(task acpruntime.Task) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessIncludeDirectories(task)
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildClaudeArgsWithIncludeDirectories(includeDirs, buildPrompt(task))
	}
	stdin, err := providercommon.JSONTaskStdin(task)
	if err != nil {
		return providercommon.CommandSpec{}, err
	}
	return providercommon.CommandSpec{
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Stdin:       stdin,
		Dir:         strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a claudeAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderClaudeCode)
}

func (a claudeAdapter) ActivityPolicy(_ acpruntime.Task) providercommon.ActivityPolicy {
	return providercommon.ActivityPolicy{
		MonitorArtifacts: true,
	}
}

func (a claudeAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop: true,
	}
}

func (a claudeAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func buildDefaultClaudeArgs(task acpruntime.Task, prompt string) []string {
	return buildClaudeArgsWithIncludeDirectories(acpruntime.ResolveHeadlessIncludeDirectories(task), prompt)
}

func buildClaudeArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	args := []string{"--output-format", "json", "--permission-mode", "bypassPermissions"}
	for _, dir := range includeDirs {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderClaudeCode, task)
}

type FakeRunner struct{}

func (FakeRunner) Run(_ context.Context, task acpruntime.Task) (acpruntime.Result, error) {
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
	if err := PersistRuntimeArtifacts(task, summary, semantic, verdict); err != nil {
		return acpruntime.Result{}, err
	}
	if err := providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderClaudeCode); err != nil {
		return acpruntime.Result{}, err
	}
	return acpruntime.Result{
		Execution: acpruntime.NewExecution(task, acpruntime.ProviderClaudeCode, "fake", "succeeded", task.StartedAtUTC.UTC().Add(2*time.Second), nil),
	}, nil
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
