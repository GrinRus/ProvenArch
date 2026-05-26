package steppolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtimedrafts"
	"gopkg.in/yaml.v3"
)

func TestStepSpecificPolicyDefinesSharedDraftOnlyObligationsForStep0(t *testing.T) {
	t.Parallel()

	policy := StepSpecificPolicy("init.step0.constitution")
	required := []string{
		`Do NOT delegate to agent/subagent helpers.`,
		`Do NOT use todo_write-style planning or long plan narration.`,
		`constitution-draft.json must use the runtime draft manifest contract exactly; legacy constitution schemas are forbidden.`,
		`This is a draft-only step; do not invent semantic entities, edges, findings, or questions on stdout.`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected step0 policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesSharedCollectRepairSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/payments",
		ReadContextRoots: []string{"/tmp/repos/payments"},
		RepoScopes:       []string{"payments"},
		PathScopes:       []string{"payments"},
		ShardID:          "payments",
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
		StartedAtUTC: time.Date(2026, 4, 21, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write ONLY inside write_root.`,
		`Suggested collect authored doc path for this shard:`,
		`Early pair-write requirement: write the suggested overview doc and shard-pack-manifest.json as one focused artifact pair`,
		`Minimal collect target shape: write "payments-overview.md" + "shard-pack-manifest.json" early`,
		`Do not wait for a complete broad repository sweep before writing shard-pack-manifest.json`,
		`TASK-SPECIFIC COLLECT MANIFEST JSON SKELETON: use the heredoc JSON embedded in the first-action command section above`,
		`COLLECT MANIFEST CONTRACT CHECKLIST:`,
		`The task-specific collect manifest JSON skeleton above is normative`,
		`Do not exit after writing markdown only; every collect shard must finish with a valid shard-pack-manifest.json.`,
		`After the first filesystem write inside write_root, stop broad repository exploration; only minimal manifest/JSON repair is allowed afterwards.`,
		`After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
		`Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
		`If authored docs and shard-pack-manifest.json already exist in write_root, stop and exit successfully.`,
		`documents[].path MUST be artifact_root-relative only`,
		`semantic.entities[*].provenance.evidence[*], semantic.edges[*].provenance.evidence[*], and semantic.findings[*].provenance.evidence[*] item MUST include non-empty repo and path values`,
		`Citation-only semantic evidence objects are forbidden`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected collect doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
	for _, forbidden := range []string{
		`FIRST COLLECT ARTIFACT PAIR COMMAND:`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md' <<'ACP_COLLECT_DOC'`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json' <<'ACP_MANIFEST_JSON'`,
	} {
		if strings.Contains(policy, forbidden) {
			t.Fatalf("doc-first collect policy must not duplicate first-action command %q:\n%s", forbidden, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDefersCollectEntrypointHintsUntilFirstPair(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repoRoot := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	for _, name := range []string{"README.md", "Makefile"} {
		if err := os.WriteFile(filepath.Join(repoRoot, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        filepath.Join(root, "workspace"),
		WriteRoot:        filepath.Join(root, "workspace", "reports", "taskruns", "run-1", "staging", "shards", "root-files"),
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/root-files",
		ReadContextRoots: []string{repoRoot},
		RepoScopes:       []string{"repo"},
		PathScopes:       []string{"README.md", "Makefile"},
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
	}

	policy := DocFirstFilesystemPolicy(task)
	if !strings.Contains(policy, "Existing repo entrypoint hints (after the first collect artifact pair exists, read only these first when further evidence is needed):") {
		t.Fatalf("collect entrypoint hints must be deferred until after first pair write, got:\n%s", policy)
	}
	if strings.Contains(policy, "Existing repo entrypoint hints (read only these first when relevant):") {
		t.Fatalf("collect entrypoint hints must not instruct provider to read before first pair write, got:\n%s", policy)
	}
}

func TestCollectFirstActionSectionWritesExactPair(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		WriteRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/payments",
		RepoScopes:   []string{"payments"},
		PathScopes:   []string{"payments"},
		ShardID:      "payments",
	}

	section := CollectFirstActionSection(task)
	required := []string{
		`COLLECT FIRST-ACTION ARTIFACT PAIR:`,
		`FIRST COLLECT ARTIFACT PAIR COMMAND:`,
		`Run this exact command as the next filesystem action after checking whether both target files already exist`,
		`do not call read_file, list_directory, grep_search, glob, find, rg, or any repository exploration before this command`,
		`The embedded skeleton is intentionally valid before additional evidence; do not improve or rewrite it before the first pair exists.`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md' <<'ACP_COLLECT_DOC'`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json' <<'ACP_MANIFEST_JSON'`,
		`"path": "payments-overview.md"`,
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST COLLECT ARTIFACT PAIR COMMAND:"); got != 1 {
		t.Fatalf("expected one first-action command heading, got %d:\n%s", got, section)
	}
}

func TestDocFirstFilesystemPolicyAddsRootFileShardHint(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/root-files",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/root-files",
		ReadContextRoots: []string{"/tmp/repos/bank"},
		RepoScopes:       []string{"bank"},
		PathScopes:       []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ExpectedArtifacts: []string{
			"shard-pack-manifest.json",
		},
		StartedAtUTC: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Root-file collect shard detected: path_scopes contains root-level files only:`,
		`.gitignore, LICENSE, Makefile, README.md, pom.xml`,
		`read only the listed root files first; do not recursively sweep top-level directories`,
		`Produce one concise root overview document in write_root, then write shard-pack-manifest.json`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected root-file shard policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDoesNotTreatTopLevelDirsAsRootFileShard(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:           "task-1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        "/tmp/workspace",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/source-docs",
		ArtifactRoot:     "reports/taskruns/run-1/staging/shards/source-docs",
		ReadContextRoots: []string{"/tmp/repos/bank"},
		RepoScopes:       []string{"bank"},
		PathScopes:       []string{"docs", "src"},
		ShardID:          "bank-source-docs",
	}

	policy := DocFirstFilesystemPolicy(task)
	if strings.Contains(policy, "Root-file collect shard detected") {
		t.Fatalf("top-level directory scopes must not receive root-file shard hint:\n%s", policy)
	}
	if !strings.Contains(policy, `Suggested collect authored doc path for this shard: "bank-source-docs-overview.md"`) {
		t.Fatalf("expected non-root shard to use shard-based doc suggestion, got:\n%s", policy)
	}

	task.PathScopes = []string{".github", "README.md"}
	task.ShardID = "bank-root-and-ci"
	policy = DocFirstFilesystemPolicy(task)
	if strings.Contains(policy, "Root-file collect shard detected") {
		t.Fatalf("mixed root files and top-level service dirs must not receive root-file shard hint:\n%s", policy)
	}
}

func TestCollectManifestTaskSkeletonParsesAsShardPackManifest(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/source-docs",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{"src"},
		ShardID:      "bank-source-docs",
		DomainID:     "payments",
		AgentRole:    "shard-analyst",
	}

	raw := CollectManifestTaskSkeleton(task, []string{"docs/overview.md", "overview.md"}, []string{"src/main.go"})
	manifest, err := contracts.ParseShardPackManifest([]byte(raw))
	if err != nil {
		t.Fatalf("expected task skeleton to parse as a valid shard pack manifest, got %v\n%s", err, raw)
	}
	if got, want := len(manifest.Documents), 2; got != want {
		t.Fatalf("documents = %d, want %d in skeleton:\n%s", got, want, raw)
	}
	if got, want := manifest.Citations[0].Path, "src/main.go"; got != want {
		t.Fatalf("citation path = %q, want %q", got, want)
	}
	if strings.Contains(raw, "scaffold") {
		t.Fatalf("collect manifest skeleton should avoid scaffold wording in provider-authored artifacts:\n%s", raw)
	}
	for _, forbidden := range []string{
		"Record concrete",
		"first citation surface",
		"before exit",
		"before exiting",
		"owner mappings if absent",
	} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("collect manifest skeleton should not persist authoring instruction %q:\n%s", forbidden, raw)
		}
	}
}

func TestRefreshCollectManifestTaskSkeletonMatchesRefreshPolicyMinimums(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/root-files",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ShardID:      "bank-root-files",
		DomainID:     "bank",
		AgentRole:    "shard-analyst",
	}

	raw := CollectManifestTaskSkeleton(task, []string{"root-overview.md"}, nil)
	manifest, err := contracts.ParseShardPackManifest([]byte(raw))
	if err != nil {
		t.Fatalf("expected refresh task skeleton to parse as a valid shard pack manifest, got %v\n%s", err, raw)
	}
	if got := len(manifest.Semantic.Coverage.Missing); got < 3 {
		t.Fatalf("refresh coverage.missing length = %d, want >= 3 in skeleton:\n%s", got, raw)
	}
	if got := len(manifest.Semantic.Questions); got < 1 {
		t.Fatalf("refresh questions length = %d, want >= 1 in skeleton:\n%s", got, raw)
	}
	question := manifest.Semantic.Questions[0]
	if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.Text) == "" {
		t.Fatalf("refresh skeleton question must include id and text, got %+v in:\n%s", question, raw)
	}
	if got, want := manifest.Citations[0].Path, "README.md"; got != want {
		t.Fatalf("root-file citation path = %q, want %q", got, want)
	}
}

func TestCollectManifestTaskSkeletonPrefersUsefulRootEvidenceWithRepairCandidates(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/root-files",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ShardID:      "bank-root-files",
		DomainID:     "bank",
		AgentRole:    "shard-analyst",
	}

	raw := CollectManifestTaskSkeleton(
		task,
		[]string{"root-overview.md"},
		[]string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
	)
	manifest, err := contracts.ParseShardPackManifest([]byte(raw))
	if err != nil {
		t.Fatalf("expected repair skeleton to parse as a valid shard pack manifest, got %v\n%s", err, raw)
	}
	if got, want := manifest.Citations[0].Path, "README.md"; got != want {
		t.Fatalf("root-file repair citation path = %q, want %q", got, want)
	}
}

func TestCollectEarlyPairWriteCommandPrefersUsefulRootEvidence(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/root-files",
		WriteRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/shards/root-files",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{".gitignore", "LICENSE", "Makefile", "README.md", "pom.xml"},
		ShardID:      "bank-root-files",
		DomainID:     "bank",
		AgentRole:    "shard-analyst",
	}

	command := CollectEarlyPairWriteCommand(task)
	for _, needle := range []string{
		"Primary scoped evidence path: `README.md`",
		`"path": "README.md"`,
		`"questions": [`,
		`"coverage": {`,
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected early pair command to contain %q, got:\n%s", needle, command)
		}
	}
	if strings.Contains(command, "Primary scoped evidence path: `.gitignore`") || strings.Contains(command, "Primary evidence path: `.gitignore`") {
		t.Fatalf("root-file shard should prefer README.md over .gitignore evidence, got:\n%s", command)
	}
}

func TestCollectEarlyPairWriteCommandAvoidsAuthoringInstructionsInArtifacts(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/source-docs",
		WriteRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/shards/source-docs",
		RepoScopes:   []string{"bank"},
		PathScopes:   []string{"src"},
		ShardID:      "bank-source-docs",
		DomainID:     "payments",
		AgentRole:    "shard-analyst",
	}

	command := CollectEarlyPairWriteCommand(task)
	for _, forbidden := range []string{
		"Record concrete",
		"first citation surface",
		"before exit",
		"before exiting",
		"owner mappings if absent",
	} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("collect early pair command should not persist authoring instruction %q:\n%s", forbidden, command)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalValidatorVerdictSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step3.findings",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/validator",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"validator-verdict.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write validator-verdict.json in write_root.`,
		`Absolute validator verdict target: "/tmp/workspace/reports/taskruns/run-1/validator/validator-verdict.json".`,
		`validator-verdict.json MUST include version=1, run_id, generated_at, verdict, and checked_paths.`,
		`issues[] items MUST use exactly the canonical validator issue shape`,
		`Do NOT put legacy finding-shaped fields inside issues[]`,
		`Canonical validator-verdict fragment below is normative`,
		`"generated_at": "2026-04-16T12:00:02Z"`,
		`"code": "staged_index_missing"`,
		`"repo": "payments-service"`,
		`"verdict": "PASS"`,
		`owner-only residual evidence gaps may still return verdict=PASS when no technical validator issues remain.`,
		`"title": "Owner mapping remains unresolved"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected findings doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestValidatorFirstActionSectionWritesExactVerdictFirst(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step3.findings",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots: []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
	}

	section := ValidatorFirstActionSection(task)
	required := []string{
		`VALIDATOR FIRST-ACTION ARTIFACT:`,
		`FIRST VALIDATOR VERDICT COMMAND:`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/validator/validator-verdict.json' <<'ACP_VALIDATOR_VERDICT_JSON'`,
		`"run_id": "run-1"`,
		`"verdict": "PASS"`,
		`"checked_paths": [`,
		`"issues": []`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected validator first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST VALIDATOR VERDICT COMMAND:"); got != 1 {
		t.Fatalf("expected first validator command once, got %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_VALIDATOR_VERDICT_JSON"); got != 2 {
		t.Fatalf("expected one validator verdict heredoc, got delimiter count %d:\n%s", got, section)
	}
}

func TestConstitutionFirstActionSectionWritesExactDraftSetFirst(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		StepContract:      "constitution",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/constitution",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"constitution-draft.json", "charter-overview.md", "baseline-subagents.yaml"},
	}

	section := ConstitutionFirstActionSection(task)
	required := []string{
		`CONSTITUTION FIRST-ACTION DRAFT ARTIFACTS:`,
		`FIRST CONSTITUTION DRAFT COMMAND:`,
		`write_root='/tmp/workspace/reports/taskruns/run-1/constitution'`,
		`draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'`,
		`cat > "$write_root/constitution-draft.json" <<'ACP_DRAFT_MANIFEST_JSON'`,
		`cat > "$draft_root/charter-overview.md" <<'ACP_DRAFT_FILE'`,
		`cat > "$draft_root/baseline-subagents.yaml" <<'ACP_DRAFT_FILE'`,
		`agents:`,
		`id: domain-analyst`,
		`id: architect-aggregator`,
		`id: system-analyst-qa`,
		`"run_id": "run-1"`,
		`"step_id": "init.step0.constitution"`,
		`"step_contract": "constitution"`,
		`"canonical_path": "charter/overview.md"`,
		`"canonical_path": "skills/subagents.yaml"`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected constitution first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST CONSTITUTION DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected constitution first-action command once, got %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one constitution manifest heredoc, got delimiter count %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_FILE"); got != 4 {
		t.Fatalf("expected two constitution draft file heredocs, got delimiter count %d:\n%s", got, section)
	}
	if strings.Contains(section, "# Baseline Subagents") {
		t.Fatalf("constitution first action must write YAML subagents, not markdown:\n%s", section)
	}

	var bundle struct {
		Agents []struct {
			ID     string   `yaml:"id"`
			Skills []string `yaml:"skills"`
		} `yaml:"agents"`
	}
	subagents := runtimeDraftFirstActionFileTemplate(task, runtimedrafts.Output{
		Path:          "baseline-subagents.yaml",
		CanonicalPath: "skills/subagents.yaml",
		Kind:          "bundle",
		Title:         "Baseline Subagents",
	})
	if err := yaml.Unmarshal([]byte(subagents), &bundle); err != nil {
		t.Fatalf("subagents first-action draft must parse as YAML: %v\n%s", err, subagents)
	}
	if got, want := len(bundle.Agents), 3; got != want {
		t.Fatalf("subagents agent count = %d, want %d in:\n%s", got, want, subagents)
	}
}

func TestAsIsFirstActionSectionWritesExactDraftSetFirst(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step2.asis_docs",
		StepContract:      "as_is",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/asis",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
	}

	section := AsIsFirstActionSection(task)
	required := []string{
		`AS-IS FIRST-ACTION DRAFT ARTIFACTS:`,
		`FIRST AS-IS DRAFT COMMAND:`,
		`write_root='/tmp/workspace/reports/taskruns/run-1/asis'`,
		`draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'`,
		`cat > "$write_root/asis-draft-manifest.json" <<'ACP_DRAFT_MANIFEST_JSON'`,
		`cat > "$draft_root/overview.md" <<'ACP_DRAFT_FILE'`,
		`cat > "$draft_root/summary.md" <<'ACP_DRAFT_FILE'`,
		`cat > "$draft_root/architect-summary.md" <<'ACP_DRAFT_FILE'`,
		`"run_id": "run-1"`,
		`"step_id": "init.step2.asis_docs"`,
		`"step_contract": "as_is"`,
		`"canonical_path": "reports/as-is/overview.md"`,
		`"canonical_path": "reports/coverage/summary.md"`,
		`"canonical_path": "reports/agent-outputs/architect/summary.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected as-is first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST AS-IS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected as-is first-action command once, got %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one as-is manifest heredoc, got delimiter count %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_FILE"); got != 6 {
		t.Fatalf("expected three as-is draft file heredocs, got delimiter count %d:\n%s", got, section)
	}
}

func TestProposalsFirstActionSectionWritesExactDraftSetFirst(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		StepContract:      "proposals",
		AgentRole:         "architect",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/proposals",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
	}

	section := ProposalsFirstActionSection(task)
	required := []string{
		`PROPOSALS FIRST-ACTION DRAFT ARTIFACTS:`,
		`FIRST PROPOSALS DRAFT COMMAND:`,
		`write_root='/tmp/workspace/reports/taskruns/run-1/proposals'`,
		`draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'`,
		`cat > "$write_root/proposals-draft-manifest.json" <<'ACP_DRAFT_MANIFEST_JSON'`,
		`cat > "$draft_root/proposal.md" <<'ACP_DRAFT_FILE'`,
		`cat > "$draft_root/changelog.md" <<'ACP_DRAFT_FILE'`,
		`"run_id": "run-1"`,
		`"step_id": "init.step4.proposals"`,
		`"step_contract": "proposals"`,
		`"canonical_path": "proposals/runtime-recommendations.md"`,
		`"canonical_path": "reports/changelog/runtime-proposals.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected proposals first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST PROPOSALS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected proposals first-action command once, got %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_MANIFEST_JSON"); got != 2 {
		t.Fatalf("expected one proposals manifest heredoc, got delimiter count %d:\n%s", got, section)
	}
	if got := strings.Count(section, "ACP_DRAFT_FILE"); got != 4 {
		t.Fatalf("expected two proposals draft file heredocs, got delimiter count %d:\n%s", got, section)
	}
}

func TestValidatorVerdictTaskSkeletonParsesWithCanonicalIssueShape(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step3.findings",
		ReadContextRoots: []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
	}

	raw := ValidatorVerdictTaskSkeleton(task)
	verdict, err := contracts.ParseValidatorVerdict([]byte(raw))
	if err != nil {
		t.Fatalf("expected validator verdict skeleton to parse, got %v\n%s", err, raw)
	}
	if verdict.RunID != "run-1" || verdict.Verdict != "PASS" {
		t.Fatalf("unexpected validator verdict skeleton: %+v", verdict)
	}
	if len(verdict.Issues) != 0 {
		t.Fatalf("expected empty canonical issues skeleton, got %+v", verdict.Issues)
	}
}

func TestValidatorVerdictTaskSkeletonIncludesCrossRepoSignalForMultiRepo(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	courseDiscovery := filepath.Join(repoRoot, "course-discovery")
	frontendPlatform := filepath.Join(repoRoot, "frontend-platform")
	if err := os.MkdirAll(courseDiscovery, 0o755); err != nil {
		t.Fatalf("mkdir course-discovery: %v", err)
	}
	if err := os.MkdirAll(frontendPlatform, 0o755); err != nil {
		t.Fatalf("mkdir frontend-platform: %v", err)
	}
	if err := os.WriteFile(filepath.Join(courseDiscovery, "README.rst"), []byte("Course discovery\n"), 0o644); err != nil {
		t.Fatalf("write course-discovery readme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frontendPlatform, "README.md"), []byte("Frontend platform\n"), 0o644); err != nil {
		t.Fatalf("write frontend-platform readme: %v", err)
	}

	task := acpruntime.Task{
		RunID:      "run-1",
		StepID:     "refresh.step3.findings",
		RepoScopes: []string{"frontend-platform", "course-discovery", "course-discovery"},
		ReadContextRoots: []string{
			"/tmp/workspace/reports/taskruns/run-1/staging/final",
			courseDiscovery,
			frontendPlatform,
		},
	}

	raw := ValidatorVerdictTaskSkeleton(task)
	verdict, err := contracts.ParseValidatorVerdict([]byte(raw))
	if err != nil {
		t.Fatalf("expected validator verdict skeleton to parse, got %v\n%s", err, raw)
	}
	if len(verdict.Findings) != 1 {
		t.Fatalf("expected one cross-repo finding, got %+v", verdict.Findings)
	}
	if len(verdict.Questions) != 1 {
		t.Fatalf("expected one cross-repo question, got %+v", verdict.Questions)
	}
	if len(verdict.Issues) != 0 {
		t.Fatalf("expected cross-repo semantic signal to stay out of technical issues, got %+v", verdict.Issues)
	}
	finding := verdict.Findings[0]
	if finding.ID != "finding.cross_repo.semantic_signal.required" {
		t.Fatalf("unexpected finding id: %+v", finding)
	}
	if len(finding.RelatedIDs) != 2 {
		t.Fatalf("expected deduplicated related repo scopes, got %+v", finding.RelatedIDs)
	}
	evidenceByRepo := map[string]string{}
	for _, item := range finding.Provenance.Evidence {
		evidenceByRepo[item.Repo] = item.Path
	}
	if evidenceByRepo["course-discovery"] != "README.rst" {
		t.Fatalf("expected course-discovery README.rst evidence, got %+v", evidenceByRepo)
	}
	if evidenceByRepo["frontend-platform"] != "README.md" {
		t.Fatalf("expected frontend-platform README.md evidence, got %+v", evidenceByRepo)
	}
	if len(verdict.Questions[0].RelatedIDs) != 2 {
		t.Fatalf("expected question to relate both repo scopes, got %+v", verdict.Questions[0])
	}
}

func TestValidatorVerdictTaskSkeletonUsesStagedFinalRootNotWriteRoot(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:     "run-1",
		StepID:    "init.step3.findings",
		WriteRoot: "/tmp/workspace/reports/taskruns/run-1/validator",
		ReadContextRoots: []string{
			"/tmp/workspace/reports/taskruns/run-1/validator",
			"/tmp/workspace",
			"/tmp/workspace/reports/taskruns/run-1/staging/final",
			"/tmp/repos/bank",
		},
	}

	raw := ValidatorVerdictTaskSkeleton(task)
	verdict, err := contracts.ParseValidatorVerdict([]byte(raw))
	if err != nil {
		t.Fatalf("expected validator verdict skeleton to parse, got %v\n%s", err, raw)
	}
	if len(verdict.CheckedPaths) != 2 {
		t.Fatalf("checked_paths = %#v, want final-run-index and citation-index", verdict.CheckedPaths)
	}
	for _, checkedPath := range verdict.CheckedPaths {
		if strings.Contains(checkedPath, "/validator/") {
			t.Fatalf("validator repair skeleton must not point checked_paths at write_root, got %#v", verdict.CheckedPaths)
		}
		if !strings.Contains(checkedPath, "/staging/final/") {
			t.Fatalf("validator repair skeleton must point checked_paths at staged final artifacts, got %#v", verdict.CheckedPaths)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalAsIsDraftSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step2.asis_docs",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/step2",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write asis-draft-manifest.json in write_root.`,
		`Write overview.md, summary.md, and architect-summary.md only under draft_final_root.`,
		`Use the FIRST AS-IS DRAFT COMMAND skeleton above as the first draft artifact set`,
		`Use staged final evidence from read_context_roots only; do NOT read sibling baseline workspaces`,
		`asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[].`,
		`"step_contract": "as_is"`,
		`"canonical_path": "reports/as-is/overview.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected as-is doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
	if strings.Contains(policy, "FIRST AS-IS DRAFT COMMAND:") {
		t.Fatalf("doc-first as-is policy must reference but not duplicate first-action command:\n%s", policy)
	}
}

func TestStepSpecificPolicyDefinesProposalsDraftContract(t *testing.T) {
	t.Parallel()

	policy := StepSpecificPolicy("init.step4.proposals")
	required := []string{
		`STEP POLICY init.step4.proposals:`,
		`Use validated staged final evidence from read_context_roots`,
		`Treat schemas/*, docs/spec/*, and the enforced prompt contract as the only manifest source of truth for proposals-draft-manifest.json.`,
		`Keep step_contract exactly "proposals"`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/*.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected proposals step policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestDocFirstFilesystemPolicyDefinesCanonicalProposalsDraftSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:            "task-1",
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		Workspace:         "/tmp/workspace",
		WriteRoot:         "/tmp/workspace/reports/taskruns/run-1/step4",
		DraftFinalRoot:    "/tmp/workspace/reports/taskruns/run-1/staging/final",
		ReadContextRoots:  []string{"/tmp/workspace/reports/taskruns/run-1/staging/final"},
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
		StartedAtUTC:      time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
	}

	policy := DocFirstFilesystemPolicy(task)
	required := []string{
		`Write proposals-draft-manifest.json in write_root.`,
		`Allowed canonical targets are proposals/* and reports/changelog/*.`,
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, optional summary, and outputs[].`,
		`Do NOT add legacy top-level fields such as pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps.`,
		`"step_contract": "proposals"`,
		`"canonical_path": "proposals/proposal-baseline/proposal.md"`,
		`"canonical_path": "reports/changelog/run-1.md"`,
	}
	for _, needle := range required {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected proposals doc-first policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestCollectArtifactRepairHintsBanLegacyRepairSurface(t *testing.T) {
	t.Parallel()

	hints := strings.Join(CollectArtifactRepairHints("manifest drift"), "\n")
	if !strings.Contains(hints, `Repair mode is artifact-only: do not invent extra repository file reads/writes after authored docs already exist.`) {
		t.Fatalf("expected collect repair hints to ban extra repair writes outside the artifact-only surface, got:\n%s", hints)
	}
}

func TestDraftArtifactRepairHintsBanLegacyRepairSurface(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:             "run-1",
		StepID:            "init.step4.proposals",
		WriteRoot:         "/tmp/write-root",
		DraftFinalRoot:    "/tmp/draft-root",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
	}

	hints := strings.Join(DraftArtifactRepairHints(task, nil), "\n")
	if !strings.Contains(hints, `Repair mode is draft-only: do not invent extra repository file reads/writes after draft files already exist.`) {
		t.Fatalf("expected draft repair hints to ban extra repair writes outside the draft-only surface, got:\n%s", hints)
	}
	for _, needle := range []string{
		`step_contract MUST be exactly "proposals"`,
		`outputs[].canonical_path values are allowed only under proposals/* or reports/changelog/* and MUST be unique.`,
		`pipeline, step, generated_at, domain_id, proposals, info_findings_noted, or orphan_coverage_gaps`,
	} {
		if !strings.Contains(hints, needle) {
			t.Fatalf("expected proposals repair hints to contain %q, got:\n%s", needle, hints)
		}
	}
}

func TestCollectPolicyRequiresStableQuestionAndFindingFields(t *testing.T) {
	t.Parallel()

	policy := StepSpecificPolicy("refresh.step1.collect") + "\n" + DocFirstFilesystemPolicy(acpruntime.Task{
		StepID:       "refresh.step1.collect",
		WriteRoot:    "/tmp/write-root",
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/openedx",
		RepoScopes:   []string{"openedx-platform", "frontend-platform"},
		PathScopes:   []string{"openedx", "frontend"},
	})
	for _, needle := range []string{
		`semantic.questions[*] must use id + text`,
		`semantic.findings[*] must use id + severity + title + description + provenance`,
		`If refresh evidence spans multiple repositories, encode at least one semantic edge, finding, or question`,
		`For multi-repo evidence, include at least one semantic edge or a finding/question`,
	} {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected collect policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestWorkspacePromptPackSectionLoadsEditableContentLayer(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	packPath := filepath.Join(workspaceDir, "skills", "prompt-packs")
	if err := os.MkdirAll(packPath, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packPath, "collect-context.md"), []byte("Collect pack line A\nCollect pack line B\n"), 0o644); err != nil {
		t.Fatalf("write prompt pack: %v", err)
	}

	section := WorkspacePromptPackSection(acpruntime.Task{
		StepID:    "init.step1.collect",
		Workspace: workspaceDir,
	})
	required := []string{
		`WORKSPACE PROMPT PACK CONTENT LAYER:`,
		`Source file: "skills/prompt-packs/collect-context.md"`,
		`editable content layer only`,
		`Collect pack line A`,
		`END WORKSPACE PROMPT PACK`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected workspace prompt-pack section to contain %q, got:\n%s", needle, section)
		}
	}
}

func TestQAPolicyUsesContextPackAndAnswerOnly(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		RunID:           "run-qa-1",
		StepID:          acpruntime.StepIDQAAsk,
		Question:        "Who owns payments-service?",
		ContextPackPath: "/tmp/workspace/reports/taskruns/run-qa-1/qa/context-pack.json",
		WriteRoot:       "/tmp/workspace/reports/taskruns/run-qa-1/qa",
	}
	policy := StepSpecificPolicy(task.StepID) + "\n" + QAFirstActionSection(task) + "\n" + DocFirstFilesystemPolicy(task)
	for _, needle := range []string{
		`STEP POLICY qa.ask:`,
		`Answer the user question only from the provided QA context pack.`,
		`Do NOT inspect source repositories, reports/taskruns history, raw logs, or sibling workspaces.`,
		`FIRST QA ANSWER COMMAND:`,
		`question = "Who owns payments-service?"`,
		`context_pack_path = "/tmp/workspace/reports/taskruns/run-qa-1/qa/context-pack.json"`,
		`qa_answer_path = "/tmp/workspace/reports/taskruns/run-qa-1/qa/qa-answer.json"`,
	} {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected qa policy to contain %q, got:\n%s", needle, policy)
		}
	}
	if !strings.Contains(policy, `"question": "Who owns payments-service?"`) {
		t.Fatalf("expected qa canonical example to include question, got:\n%s", policy)
	}
}

func TestCollectRepoEntrypointHintsIncludesOwnershipFiles(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".github"), 0o755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".github", "CODEOWNERS"), []byte("* @platform\n"), 0o644); err != nil {
		t.Fatalf("write codeowners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "CODEOWNERS"), []byte("* @root-owners\n"), 0o644); err != nil {
		t.Fatalf("write root codeowners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "OWNERS.md"), []byte("# Owners\n"), 0o644); err != nil {
		t.Fatalf("write owners: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "MAINTAINERS.yaml"), []byte("maintainers: []\n"), 0o644); err != nil {
		t.Fatalf("write maintainers: %v", err)
	}

	hints := CollectRepoEntrypointHints(acpruntime.Task{
		StepID:           "init.step1.collect",
		ReadContextRoots: []string{repoRoot},
	})
	joined := strings.Join(hints, "\n")
	for _, needle := range []string{".github/CODEOWNERS", "CODEOWNERS", "OWNERS.md", "MAINTAINERS.yaml"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected entrypoint hints to contain %q, got:\n%s", needle, joined)
		}
	}
}
