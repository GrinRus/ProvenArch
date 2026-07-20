package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/model"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestRefreshCollectSemanticGuardFiltersRuntimeMetadataAndOffScopeCandidates(t *testing.T) {
	t.Parallel()

	semantic := refreshGuardFixtureSemantic()
	task := acpruntime.Task{
		StepID:     "refresh.step1.collect",
		RepoScopes: []string{"payments-service"},
	}

	guarded, warnings := guardRefreshCollectSemantic(task.StepID, task, semantic)

	assertSemanticEntityIDs(t, guarded.Entities, []string{"svc.payments", "svc.payments-worker"})
	assertSemanticEdgeIDs(t, guarded.Edges, []string{"edge.payments.calls.worker"})
	assertSemanticFindingIDs(t, guarded.Findings, []string{
		"finding.payments.slo",
		"semantic_guard.refresh.off-scope.entity-svc-crm",
		"semantic_guard.refresh.runtime-metadata.entity-runtime-provider-claude-code",
	})
	assertSemanticQuestionIDs(t, guarded.Questions, []string{"q.payments.slo"})

	for _, warning := range []string{
		"semantic_guard: runtime_metadata_filtered in refresh.step1.collect candidate_type=entity candidate_id=runtime.provider.claude-code",
		"semantic_guard: off_scope_filtered in refresh.step1.collect candidate_type=entity candidate_id=svc.crm",
	} {
		if !containsString(warnings, warning) {
			t.Fatalf("expected warning %q in %#v", warning, warnings)
		}
	}
}

func TestInitCollectSemanticGuardIsNoop(t *testing.T) {
	t.Parallel()

	semantic := refreshGuardFixtureSemantic()
	task := acpruntime.Task{
		StepID:     "init.step1.collect",
		RepoScopes: []string{"payments-service"},
	}

	guarded, warnings := guardRefreshCollectSemantic(task.StepID, task, semantic)
	if len(warnings) != 0 {
		t.Fatalf("expected init guard to be silent, got %#v", warnings)
	}
	if !reflect.DeepEqual(guarded, semantic) {
		t.Fatalf("expected init semantic payload to remain unchanged\nwant=%#v\ngot=%#v", semantic, guarded)
	}
}

func TestApplyCollectRuntimeExecutionUsesGuardedRefreshSemantic(t *testing.T) {
	t.Parallel()

	ws := createWorkspace(t)
	if err := ws.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	writeRoot := t.TempDir()
	manifest := contracts.ShardPackManifest{
		Version:      1,
		RunID:        "run-refresh",
		StepID:       "refresh.step1.collect",
		ShardID:      "payments",
		DomainID:     "payments-service",
		AgentRole:    "shard-analyst",
		ArtifactRoot: "reports/taskruns/run-refresh/staging/shards/payments",
		RepoScopes:   []string{"payments-service"},
		PathScopes:   []string{"."},
		Summary:      "payments refresh evidence",
		Documents: []contracts.AuthoredDocument{
			{
				ID:            "doc.payments",
				Kind:          "report",
				Title:         "Payments refresh",
				Path:          "overview.md",
				CanonicalPath: "reports/as-is/payments/overview.md",
				Topics:        []string{"payments"},
				CitationIDs:   []string{"cite.payments"},
			},
		},
		Citations: []contracts.DocumentCitation{
			{
				ID:          "cite.payments",
				Repo:        "payments-service",
				Path:        "README.md",
				ClaimIDs:    []string{"claim.payments"},
				DocumentIDs: []string{"doc.payments"},
			},
		},
		Semantic: refreshGuardFixtureSemantic(),
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, shardPackManifestFile), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	execution := &pipelineExecution{
		workspace: ws,
		store:     model.NewStore(ws),
	}
	task := acpruntime.Task{
		TaskID:       "task-run-refresh-step1-collect-payments",
		RunID:        "run-refresh",
		StepID:       "refresh.step1.collect",
		ShardID:      "payments",
		DomainID:     "payments-service",
		ArtifactRoot: "reports/taskruns/run-refresh/staging/shards/payments",
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"payments-service"},
		PathScopes:   []string{"."},
	}

	result, err := execution.applyCollectRuntimeExecution(
		task.StepID,
		task.DomainID,
		task,
		contracts.RuntimeExecution{Warnings: nil},
		"fake",
		"fake",
	)
	if err != nil {
		t.Fatalf("apply collect runtime execution: %v", err)
	}
	if result.Apply.UpsertedEntities != 2 {
		t.Fatalf("expected two guarded entities to be applied, got %+v", result.Apply)
	}
	if len(execution.shardPacks) != 1 {
		t.Fatalf("expected one guarded shard pack, got %d", len(execution.shardPacks))
	}
	assertSemanticEntityIDs(t, execution.shardPacks[0].Semantic.Entities, []string{"svc.payments", "svc.payments-worker"})
	assertSemanticFindingIDs(t, execution.shardPacks[0].Semantic.Findings, []string{
		"finding.payments.slo",
		"semantic_guard.refresh.off-scope.entity-svc-crm",
		"semantic_guard.refresh.runtime-metadata.entity-runtime-provider-claude-code",
	})
	if _, err := os.Stat(filepath.Join(ws.Path, "model", "entities", "svc.payments.yaml")); err != nil {
		t.Fatalf("expected kept payments entity in model store: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "model", "entities", "svc.crm.yaml")); err == nil {
		t.Fatalf("off-scope entity was applied to model store")
	}
}

func refreshGuardFixtureSemantic() contracts.SemanticSnapshot {
	return contracts.SemanticSnapshot{
		Coverage: contracts.Coverage{
			Observed: []string{"payments refresh"},
			Missing:  []string{"owner mappings"},
		},
		Entities: []contracts.Entity{
			{
				ID:   "svc.payments",
				Type: "service",
				Name: "Payments API",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.9,
					Evidence:   []contracts.Evidence{{Repo: "payments-service", Path: "README.md"}},
				},
			},
			{
				ID:   "svc.payments-worker",
				Type: "service",
				Name: "Payments Worker",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "payments-service", Path: "worker/main.go"}},
				},
			},
			{
				ID:   "runtime.provider.claude-code",
				Type: "runtime_provider",
				Name: "Claude runtime provider",
				Provenance: contracts.Provenance{
					Kind:       "assertion",
					Confidence: 1,
					Evidence:   []contracts.Evidence{{Repo: "payments-service", Path: "reports/taskruns/run-refresh/raw/meta.json"}},
				},
			},
			{
				ID:   "svc.crm",
				Type: "service",
				Name: "CRM",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "crm-service", Path: "README.md"}},
				},
			},
		},
		Edges: []contracts.Edge{
			{
				ID:   "edge.payments.calls.worker",
				Type: "calls",
				From: "svc.payments",
				To:   "svc.payments-worker",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "payments-service", Path: "README.md"}},
				},
			},
			{
				ID:   "edge.payments.calls.crm",
				Type: "calls",
				From: "svc.payments",
				To:   "svc.crm",
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.7,
					Evidence:   []contracts.Evidence{{Repo: "crm-service", Path: "README.md"}},
				},
			},
		},
		Findings: []contracts.Finding{
			{
				ID:         "finding.payments.slo",
				Severity:   "medium",
				Title:      "Payments SLO missing",
				RelatedIDs: []string{"svc.payments"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "payments-service", Path: "README.md"}},
				},
			},
			{
				ID:         "finding.crm.owner",
				Severity:   "medium",
				Title:      "CRM owner missing",
				RelatedIDs: []string{"svc.crm"},
				Provenance: contracts.Provenance{
					Kind:       "observation",
					Confidence: 0.8,
					Evidence:   []contracts.Evidence{{Repo: "crm-service", Path: "README.md"}},
				},
			},
		},
		Questions: []contracts.Question{
			{ID: "q.payments.slo", Text: "Who owns payments SLO?", RelatedIDs: []string{"svc.payments"}},
			{ID: "q.crm.owner", Text: "Who owns CRM?", RelatedIDs: []string{"svc.crm"}},
		},
	}
}

func assertSemanticEntityIDs(t *testing.T, entities []contracts.Entity, want []string) {
	t.Helper()

	got := make([]string, 0, len(entities))
	for _, entity := range entities {
		got = append(got, entity.ID)
	}
	assertSortedStringsEqual(t, got, want)
}

func assertSemanticEdgeIDs(t *testing.T, edges []contracts.Edge, want []string) {
	t.Helper()

	got := make([]string, 0, len(edges))
	for _, edge := range edges {
		got = append(got, edge.ID)
	}
	assertSortedStringsEqual(t, got, want)
}

func assertSemanticFindingIDs(t *testing.T, findings []contracts.Finding, want []string) {
	t.Helper()

	got := make([]string, 0, len(findings))
	for _, finding := range findings {
		got = append(got, finding.ID)
	}
	assertSortedStringsEqual(t, got, want)
}

func assertSemanticQuestionIDs(t *testing.T, questions []contracts.Question, want []string) {
	t.Helper()

	got := make([]string, 0, len(questions))
	for _, question := range questions {
		got = append(got, question.ID)
	}
	assertSortedStringsEqual(t, got, want)
}

func assertSortedStringsEqual(t *testing.T, got []string, want []string) {
	t.Helper()

	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected strings:\nwant=%v\ngot= %v", want, got)
	}
}
