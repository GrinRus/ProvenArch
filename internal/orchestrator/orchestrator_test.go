package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/testutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRunInitPipelineMaterializesExpectedArtifacts(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if len(artifacts) == 0 {
		t.Fatalf("expected non-empty artifacts")
	}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		key := artifact.Kind + "|" + artifact.Path
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate artifact entry detected: %s", key)
		}
		seen[key] = struct{}{}
	}

	for _, rel := range []string{
		"skills/subagents.yaml",
		"model/entities/svc.payments-service.yaml",
		"reports/as-is/overview.md",
		"reports/findings/findings.md",
		"reports/changelog",
	} {
		abs := filepath.Join(ws.Path, rel)
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("expected artifact %q: %v", rel, err)
		}
	}
}

func TestInitStep0FallsBackWhenWizardContractMissing(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if !hasWarningPrefix(info.Warnings, "step0_wizard_contract_missing:") {
		t.Fatalf("expected step0 missing contract warning, got %+v", info.Warnings)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "charter/overview.md"))
	if err != nil {
		t.Fatalf("read charter overview: %v", err)
	}
	if !strings.Contains(string(content), "Generated baseline charter for ACP MVP.") {
		t.Fatalf("expected baseline overview fallback, got %q", string(content))
	}
}

func TestInitStep0FallsBackWhenWizardContractInvalid(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile(step0WizardContractPath, []byte("{invalid-json")); err != nil {
		t.Fatalf("write invalid step0 contract: %v", err)
	}

	service := NewService()
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}
	if !hasWarningPrefix(info.Warnings, "step0_wizard_contract_invalid:") {
		t.Fatalf("expected step0 invalid contract warning, got %+v", info.Warnings)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "charter/overview.md"))
	if err != nil {
		t.Fatalf("read charter overview: %v", err)
	}
	if !strings.Contains(string(content), "Generated baseline charter for ACP MVP.") {
		t.Fatalf("expected baseline overview fallback, got %q", string(content))
	}
}

func TestInitStep0UsesWizardContractDeterministically(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	step0Contract := `{
  "version": 1,
  "project_name": "Payments Platform",
  "scope": "payments, users, ci-cd",
  "nfr_priorities": ["availability", "traceability"],
  "rules": ["evidence-first findings", "no silent re-key"]
}
`
	if err := ws.WriteFile(step0WizardContractPath, []byte(step0Contract)); err != nil {
		t.Fatalf("write step0 contract: %v", err)
	}

	service := NewService()
	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}

	overviewPath := filepath.Join(ws.Path, "charter/overview.md")
	domainPath := filepath.Join(ws.Path, "charter/cards/domains/payments-service.md")
	teamPath := filepath.Join(ws.Path, "charter/cards/teams/platform.md")
	nfrPath := filepath.Join(ws.Path, "charter/nfr.yaml")
	rulesPath := filepath.Join(ws.Path, "charter/rules.yaml")

	firstOverview, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatalf("read first overview: %v", err)
	}
	firstDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read first domain card: %v", err)
	}
	firstTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read first team card: %v", err)
	}
	firstNFR, err := os.ReadFile(nfrPath)
	if err != nil {
		t.Fatalf("read first nfr: %v", err)
	}
	firstRules, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read first rules: %v", err)
	}

	if !strings.Contains(string(firstOverview), "- project_name: `Payments Platform`") {
		t.Fatalf("expected wizard project in overview, got %q", string(firstOverview))
	}
	if !strings.Contains(string(firstDomain), "- charter_scope: `payments, users, ci-cd`") {
		t.Fatalf("expected wizard scope in domain card, got %q", string(firstDomain))
	}
	if !strings.Contains(string(firstTeam), "- charter_project: `Payments Platform`") {
		t.Fatalf("expected wizard project in team card, got %q", string(firstTeam))
	}
	if !strings.Contains(string(firstNFR), "availability") || !strings.Contains(string(firstNFR), "traceability") {
		t.Fatalf("expected wizard nfr priorities in nfr.yaml, got %q", string(firstNFR))
	}
	if !strings.Contains(string(firstRules), "evidence-first findings") {
		t.Fatalf("expected wizard rules in rules.yaml, got %q", string(firstRules))
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}

	secondOverview, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatalf("read second overview: %v", err)
	}
	secondDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read second domain card: %v", err)
	}
	secondTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read second team card: %v", err)
	}
	secondNFR, err := os.ReadFile(nfrPath)
	if err != nil {
		t.Fatalf("read second nfr: %v", err)
	}
	secondRules, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read second rules: %v", err)
	}

	if string(firstOverview) != string(secondOverview) {
		t.Fatalf("overview materialization from wizard contract is not deterministic")
	}
	if string(firstDomain) != string(secondDomain) {
		t.Fatalf("domain card materialization from wizard contract is not deterministic")
	}
	if string(firstTeam) != string(secondTeam) {
		t.Fatalf("team card materialization from wizard contract is not deterministic")
	}
	if string(firstNFR) != string(secondNFR) {
		t.Fatalf("nfr materialization from wizard contract is not deterministic")
	}
	if string(firstRules) != string(secondRules) {
		t.Fatalf("rules materialization from wizard contract is not deterministic")
	}
}

func TestStartAsyncRunRegistersAndCompletesRun(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService()

	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}
	if runID == "" {
		t.Fatalf("expected run id")
	}

	info := waitForRunTerminalState(t, service, runID, 8*time.Second)
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected async run success, got status=%s error=%s", info.Status, info.Error)
	}
	if info.CurrentStep != "refresh.step4.proposals" {
		t.Fatalf("expected final current_step to point to last step, got %q", info.CurrentStep)
	}
}

func TestStartAsyncRunFailsWhenWorkspaceLayoutCannotBeCreated(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	// Make `reports` a file to force EnsureLayout failure when creating nested dirs.
	if err := os.WriteFile(filepath.Join(ws.Path, "reports"), []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("write conflicting reports file: %v", err)
	}

	service := NewService()
	runID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start async run: %v", err)
	}

	info := waitForRunTerminalState(t, service, runID, 8*time.Second)
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed run status, got %s", info.Status)
	}
	if info.Error == "" {
		t.Fatalf("expected non-empty error on failed run")
	}
}

func TestStartAsyncRunDebounceLastEventWins(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(delayedRunner{delay: 200 * time.Millisecond}),
		WithDebounceWindow(5*time.Minute),
	)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run1: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run2: %v", err)
	}
	run3, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start run3: %v", err)
	}
	if run2 == run3 {
		t.Fatalf("expected run3 to supersede run2 with a new run id")
	}

	info3 := waitForRunTerminalState(t, service, run3, 8*time.Second)
	if info3.Status != RunStatusSucceeded {
		t.Fatalf("expected run3 success, got %s (%s)", info3.Status, info3.Error)
	}

	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusFailed || info2.Error == "" {
		t.Fatalf("expected run2 superseded failure, got status=%s error=%q", info2.Status, info2.Error)
	}

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusSucceeded {
		t.Fatalf("expected run1 success, got status=%s error=%q", info1.Status, info1.Error)
	}
}

func TestStartAsyncRunRejectsWhenPendingOutsideDebounceWindow(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(delayedRunner{delay: 220 * time.Millisecond}),
		WithDebounceWindow(10*time.Millisecond),
	)

	run1, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start first run: %v", err)
	}
	run2, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("start second run: %v", err)
	}
	beforeRejectRunCount := runRegistrySize(service)

	time.Sleep(25 * time.Millisecond)
	rejectedRunID, err := service.StartAsyncRun(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected debounce-window rejection")
	}
	if rejectedRunID != "" {
		t.Fatalf("expected empty run id on rejection, got %q", rejectedRunID)
	}
	if got := runRegistrySize(service); got != beforeRejectRunCount {
		t.Fatalf("expected run registry size to remain %d after rejection, got %d", beforeRejectRunCount, got)
	}
	assertRunRegistryContainsOnly(t, service, run1, run2)

	info1 := waitForRunTerminalState(t, service, run1, 8*time.Second)
	if info1.Status != RunStatusSucceeded {
		t.Fatalf("expected run1 success, got status=%s error=%q", info1.Status, info1.Error)
	}
	info2 := waitForRunTerminalState(t, service, run2, 8*time.Second)
	if info2.Status != RunStatusSucceeded {
		t.Fatalf("expected run2 success, got status=%s error=%q", info2.Status, info2.Error)
	}
}

func TestRunInitPipelineMaterializesDocArtifactMetadata(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithRunner(docArtifactRunner{}),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init with doc artifact runner: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}

	content, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/doc-artifacts.md"))
	if err != nil {
		t.Fatalf("read doc-artifacts report: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "Doc Artifact Metadata") {
		t.Fatalf("expected metadata title in doc-artifacts report, got %q", text)
	}
	if !strings.Contains(text, "imports.architecture.notes") {
		t.Fatalf("expected doc artifact id in report, got %q", text)
	}
	if !strings.Contains(text, "docs/imports/architecture-notes.md") {
		t.Fatalf("expected doc artifact path in report, got %q", text)
	}
}

func TestInitStep1EnrichesCanonicalCardsDeterministically(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}

	domainPath := filepath.Join(ws.Path, "charter/cards/domains/payments-service.md")
	teamPath := filepath.Join(ws.Path, "charter/cards/teams/platform.md")

	firstDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read first domain card: %v", err)
	}
	firstTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read first team card: %v", err)
	}
	if !strings.Contains(string(firstDomain), "## Derived (ACP Step1)") {
		t.Fatalf("expected derived section in domain card, got %q", string(firstDomain))
	}
	if !strings.Contains(string(firstDomain), "related_entities: `svc.payments-service`") {
		t.Fatalf("expected related entity in domain card, got %q", string(firstDomain))
	}
	if !strings.Contains(string(firstTeam), "## Derived (ACP Step1)") {
		t.Fatalf("expected derived section in team card, got %q", string(firstTeam))
	}
	if !strings.Contains(string(firstTeam), "related_services: none") {
		t.Fatalf("expected deterministic related services in team card, got %q", string(firstTeam))
	}
	if !strings.Contains(string(firstTeam), "evidence_refs: none") {
		t.Fatalf("expected team evidence refs section in team card, got %q", string(firstTeam))
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}

	secondDomain, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatalf("read second domain card: %v", err)
	}
	secondTeam, err := os.ReadFile(teamPath)
	if err != nil {
		t.Fatalf("read second team card: %v", err)
	}
	if string(firstDomain) != string(secondDomain) {
		t.Fatalf("domain card enrichment is not deterministic")
	}
	if string(firstTeam) != string(secondTeam) {
		t.Fatalf("team card enrichment is not deterministic")
	}
}

func TestInitStep1ExecutesPerCanonicalDomainAndMaterializesDomainTaskruns(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	runner := &trackingRunner{}
	service := NewService(
		WithRunner(runner),
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s", info.Status)
	}

	domainIDs, err := loadCanonicalDomainIDs(ws)
	if err != nil {
		t.Fatalf("load canonical domain ids: %v", err)
	}
	if len(domainIDs) == 0 {
		t.Fatalf("expected canonical domain cards to exist after init")
	}

	collectTasks := runner.tasksForStep("init.step1.collect")
	if len(collectTasks) != len(domainIDs) {
		t.Fatalf("expected %d step1 runtime executions, got %d", len(domainIDs), len(collectTasks))
	}

	expectedScopes := collectRepoScopes(ws.Manifest.Repos)
	seenScopes := make([]string, 0, len(collectTasks))
	for _, task := range collectTasks {
		if len(task.RepoScopes) != 1 {
			t.Fatalf("expected exactly one repo scope per domain task, got %v", task.RepoScopes)
		}
		seenScopes = append(seenScopes, task.RepoScopes[0])
	}
	sort.Strings(seenScopes)
	if len(seenScopes) != len(expectedScopes) {
		t.Fatalf("unexpected scope count: got %d want %d", len(seenScopes), len(expectedScopes))
	}
	for idx := range seenScopes {
		if seenScopes[idx] != expectedScopes[idx] {
			t.Fatalf("unexpected per-domain scopes: got %v want %v", seenScopes, expectedScopes)
		}
	}

	for _, domainID := range domainIDs {
		taskrunPath := filepath.Join(
			ws.Path,
			"reports",
			"taskruns",
			fmt.Sprintf("%s-init-step1-collect-domain-%s.json", info.RunID, sanitizeDomainArtifactSlug(domainID)),
		)
		taskrun, readErr := os.ReadFile(taskrunPath)
		if readErr != nil {
			t.Fatalf("read per-domain taskrun %s: %v", taskrunPath, readErr)
		}
		var payload struct {
			Meta struct {
				StepID string `json:"step_id"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(taskrun, &payload); err != nil {
			t.Fatalf("decode per-domain taskrun %s: %v", taskrunPath, err)
		}
		if payload.Meta.StepID != "init.step1.collect" {
			t.Fatalf("expected per-domain taskrun step id init.step1.collect, got %q", payload.Meta.StepID)
		}

		domainOutputPath := filepath.Join(ws.Path, "reports", "agent-outputs", "domains", fmt.Sprintf("%s.md", domainID))
		if _, statErr := os.Stat(domainOutputPath); statErr != nil {
			t.Fatalf("expected domain output %s: %v", domainOutputPath, statErr)
		}
	}
}

func TestArchitectSummaryIsDeterministicAcrossRuns(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(
		WithClock(func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
		}),
	)

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("first init run failed: %v", err)
	}
	summaryPath := filepath.Join(ws.Path, "reports", "agent-outputs", "architect", "summary.md")
	firstSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read first architect summary: %v", err)
	}

	if _, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	}); err != nil {
		t.Fatalf("second init run failed: %v", err)
	}
	secondSummary, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read second architect summary: %v", err)
	}

	if string(firstSummary) != string(secondSummary) {
		t.Fatalf("architect summary is not deterministic across runs")
	}
	text := string(firstSummary)
	if strings.Contains(text, "run_") {
		t.Fatalf("architect summary must not contain run id markers: %q", text)
	}
	paymentsIdx := strings.Index(text, "`payments-service`")
	usersIdx := strings.Index(text, "`users-service`")
	if paymentsIdx < 0 || usersIdx < 0 {
		t.Fatalf("expected architect summary to include canonical domains, got %q", text)
	}
	if paymentsIdx > usersIdx {
		t.Fatalf("expected sorted domain order in architect summary, got %q", text)
	}
}

func TestRefreshStep1MissingCanonicalDomainsWritesQuestionWithoutAutoCreate(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	service := NewService(WithRunner(step3ParseFailureRunner{}))

	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if !strings.Contains(string(questionsReport), "q.domains.missing-canonical-cards") {
		t.Fatalf("expected missing-canonical-domains question in coverage report, got %q", string(questionsReport))
	}
	if !strings.Contains(string(questionsReport), "q.teams.missing-canonical-cards") {
		t.Fatalf("expected missing-canonical-teams question in coverage report, got %q", string(questionsReport))
	}

	matches, err := filepath.Glob(filepath.Join(ws.Path, "charter/cards/domains", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical domain cards: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("step1 must not auto-create canonical domain cards, found: %v", matches)
	}
	teamMatches, err := filepath.Glob(filepath.Join(ws.Path, "charter/cards/teams", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical team cards: %v", err)
	}
	if len(teamMatches) != 0 {
		t.Fatalf("step1 must not auto-create canonical team cards, found: %v", teamMatches)
	}
}

func TestRefreshStep1MissingRepoScopeWritesQuestion(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure workspace layout: %v", err)
	}
	if err := ws.WriteFile("charter/cards/domains/billing.md", []byte("# Domain: Billing\n\n- id: `billing`\n")); err != nil {
		t.Fatalf("write billing domain card: %v", err)
	}

	service := NewService(WithRunner(step3ParseFailureRunner{}))
	info, _, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineRefresh,
		NonInteractive: true,
	})
	if err == nil {
		t.Fatalf("expected refresh run to fail at step3")
	}
	if info.Status != RunStatusFailed {
		t.Fatalf("expected failed status, got %s", info.Status)
	}

	questionsReport, err := os.ReadFile(filepath.Join(ws.Path, "reports/coverage/open-questions.md"))
	if err != nil {
		t.Fatalf("read open questions report: %v", err)
	}
	if !strings.Contains(string(questionsReport), "q.domain.billing.missing-repo-scope") {
		t.Fatalf("expected missing repo scope question in coverage report, got %q", string(questionsReport))
	}
}

type delayedRunner struct {
	delay time.Duration
}

func (r delayedRunner) Run(ctx context.Context, task claudecode.Task) (claudecode.Result, error) {
	select {
	case <-ctx.Done():
		return claudecode.Result{}, ctx.Err()
	case <-time.After(r.delay):
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

type step3ParseFailureRunner struct{}

func (step3ParseFailureRunner) Run(ctx context.Context, task claudecode.Task) (claudecode.Result, error) {
	if strings.HasSuffix(task.StepID, "step3.findings") {
		return claudecode.Result{}, claudecode.WrapRunnerError(
			claudecode.ErrorCodeRunnerParseFailed,
			"synthetic parse failure at findings step",
			nil,
		)
	}
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (step3ParseFailureRunner) Preflight(context.Context) error {
	return nil
}

type docArtifactRunner struct{}

func (docArtifactRunner) Run(ctx context.Context, task claudecode.Task) (claudecode.Result, error) {
	base := claudecode.FakeRunner{}
	result, err := base.Run(ctx, task)
	if err != nil {
		return claudecode.Result{}, err
	}
	if strings.HasSuffix(task.StepID, "step1.collect") {
		taskResult := result.TaskResult
		taskResult.Changeset = append(taskResult.Changeset, contracts.Operation{
			Op: "add_doc_artifact",
			DocArtifact: &contracts.DocArtifact{
				ID:     "imports.architecture.notes",
				Kind:   "imported-doc",
				Title:  "Architecture Notes",
				Path:   "docs/imports/architecture-notes.md",
				Format: "markdown",
			},
		})
		raw, marshalErr := json.MarshalIndent(taskResult, "", "  ")
		if marshalErr != nil {
			return claudecode.Result{}, marshalErr
		}
		result.TaskResult = taskResult
		result.RawJSON = raw
	}
	return result, nil
}

func createWorkspace(t *testing.T) workspace.Root {
	t.Helper()

	root := t.TempDir()
	repoA := filepath.Join(root, "repos", "payments-service")
	repoB := filepath.Join(root, "repos", "users-service")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("create payments repo: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("create users repo: %v", err)
	}
	manifest := `version: 1
repos:
  - name: payments-service
    path: ` + repoA + `
  - name: users-service
    path: ` + repoB + `
`
	if err := os.WriteFile(filepath.Join(root, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	ws, err := workspace.Open(root)
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}
	return ws
}

func waitForRunTerminalState(t *testing.T, service *Service, runID string, timeout time.Duration) RunInfo {
	t.Helper()

	var terminal RunInfo
	testutil.WaitFor(t, timeout, testutil.WaitDescription("run %q did not reach terminal status", runID), func() (bool, error) {
		info, ok := service.GetRun(runID)
		if ok && (info.Status == RunStatusSucceeded || info.Status == RunStatusFailed) {
			terminal = info
			return true, nil
		}
		return false, nil
	})
	return terminal
}

func runRegistrySize(service *Service) int {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return len(service.runs)
}

func assertRunRegistryContainsOnly(t *testing.T, service *Service, runIDs ...string) {
	t.Helper()
	expected := map[string]struct{}{}
	for _, runID := range runIDs {
		expected[runID] = struct{}{}
	}

	service.mu.RLock()
	defer service.mu.RUnlock()

	if len(service.runs) != len(expected) {
		t.Fatalf("unexpected run registry size: got %d want %d", len(service.runs), len(expected))
	}
	for runID := range service.runs {
		if _, ok := expected[runID]; !ok {
			t.Fatalf("unexpected run registry entry %q", runID)
		}
	}
}

func hasWarningPrefix(warnings []string, prefix string) bool {
	for _, warning := range warnings {
		if strings.HasPrefix(strings.TrimSpace(warning), prefix) {
			return true
		}
	}
	return false
}

type trackingRunner struct {
	mu    sync.Mutex
	tasks []claudecode.Task
}

func (r *trackingRunner) Run(ctx context.Context, task claudecode.Task) (claudecode.Result, error) {
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
	return claudecode.FakeRunner{}.Run(ctx, task)
}

func (r *trackingRunner) tasksForStep(stepID string) []claudecode.Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := []claudecode.Task{}
	for _, task := range r.tasks {
		if task.StepID == stepID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}
