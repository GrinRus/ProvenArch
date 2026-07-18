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
		`Do not mention later collection steps`,
		`not confirmed in the current constitution evidence`,
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
		`The first collect filesystem work unit may contain only two mechanically simple commands: one bounded evidence read/list, then one direct literal write`,
		`Cap the bounded evidence read/list to at most 8 representative files and at most the first 6000 bytes from each file`,
		`Do not run analysis-only narration, status/progress text, todo/planning, broad repository sweeps, or any second read-only preflight before the direct literal write command.`,
		`Evidence-write pair requirement: the first work unit writes the suggested overview doc and shard-pack-manifest.json as one focused marker-free evidence-backed artifact pair`,
		`Minimal collect target shape: write "payments-overview.md" + "shard-pack-manifest.json" with concrete observed evidence`,
		`Do not wait for a complete broad repository sweep before writing shard-pack-manifest.json; the bounded first action is enough`,
		`Do not rely on focused collect repair as the expected success path; normal collect must attempt to produce the valid pair itself.`,
		`Before both target files exist, do not use Ruby, Node, Python, Perl, awk, jq, generated source-code strings, template programs, or nested quote tricks`,
		`If the direct write command fails before both targets exist, immediately retry the same direct literal write pattern with simpler content from observed evidence; do not wait for collect_pair_repair.`,
		`TASK-SPECIFIC COLLECT MANIFEST JSON SKELETON: use the JSON embedded in the collect evidence-first section above as a schema/key/type guide`,
		`COLLECT MANIFEST CONTRACT CHECKLIST:`,
		`The task-specific collect manifest JSON skeleton above is normative`,
		`Do not exit after writing markdown only; every collect shard must finish with a valid shard-pack-manifest.json.`,
		`The final collect pair must not be seed-only, scaffold-only, or copied unchanged from the skeleton`,
		`The final collect markdown must not describe itself as an initial/temporary artifact, interrupted evidence read, or content that "will be repaired"`,
		`The final collect markdown must not mention bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or runtime repair mechanics`,
		`After writing the evidence-backed pair, avoid broad repository exploration; only minimal manifest/JSON repair needed for the current shard is allowed afterwards.`,
		`After writing shard-pack-manifest.json, do NOT continue broad list_directory/read_file sweeps across repo roots.`,
		`Do NOT read reports/taskruns/**, raw runtime logs, or previously generated shard-pack-manifest.json files as schema examples during collect.`,
		`If authored docs and shard-pack-manifest.json already exist in write_root, stop only after confirming they contain marker-free scoped evidence for this shard and are not placeholder prose.`,
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
	if !strings.Contains(policy, "Existing repo entrypoint hints (read these first for the bounded collect evidence pass):") {
		t.Fatalf("collect entrypoint hints must guide the bounded evidence pass, got:\n%s", policy)
	}
	if strings.Contains(policy, "Existing repo entrypoint hints (read only these first when relevant):") {
		t.Fatalf("collect entrypoint hints must use the collect-specific bounded evidence wording, got:\n%s", policy)
	}
}

func TestCollectFirstActionSectionDefinesEvidenceFirstTargets(t *testing.T) {
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
		`COLLECT EVIDENCE-FIRST ARTIFACT PAIR:`,
		`FIRST COLLECT BOUNDED WRITE ACTION:`,
		`COLLECT FINAL WRITE REQUIREMENT:`,
		`COLLECT MANIFEST TASK SKELETON:`,
		`SKELETON USE:`,
		`start by writing an evidence-backed artifact pair; do not write a seed-only pair`,
		`next filesystem work unit may contain only two mechanically simple commands: one bounded evidence read/list, then one direct literal write of both exact targets`,
		`inspect at most 8 representative entrypoint/build/config/source files`,
		`Read at most the first 6000 bytes from any file.`,
		`Assigned path_scopes may be directories for discovery, but manifest citations and semantic provenance paths must resolve to concrete existing files, never directories.`,
		`prove every citation/provenance repo evidence path with file-level checks such as test -f, rg --files, or portable find ... -type f -print`,
		`Every citations[].id must be unique`,
		`derive each citation id from the shard/document stem plus the repo path slug`,
		`If a scoped path is missing or directory-only, record it as coverage.missing or a question instead of using it as citation/provenance evidence.`,
		`Syntax-only checks such as jq empty or python3 -m json.tool are insufficient`,
		`verify semantic.questions[] all have id and text, citations[].id has no duplicates`,
		`every citation has non-empty claim_ids and document_ids`,
		`Use python3, not python; do not use GNU-only find -printf; do not assign to the zsh-reserved status variable.`,
		`Do not emit analysis-only prose, status/progress narration, todo/planning, broad repo sweeps, or any second read-only preflight before writing the artifact pair.`,
		`Before both targets exist, do not use Ruby, Node, Python, Perl, awk, jq, generated source-code strings, template programs, or nested quote tricks`,
		`Write the authored document and shard-pack-manifest.json with direct shell heredoc/printf/tee literal content from the bounded reads in the first work unit.`,
		`If a write command fails before both targets exist, immediately retry with a simpler direct literal write; do not wait for collect_pair_repair.`,
		`Normal collect must not depend on collect_pair_repair as the expected path to success.`,
		`Final collect markdown must be operator-facing architecture evidence. Do not mention bounded reads/passes, guessed paths/files/evidence, expected-missing path checks, recovery attempts, or later repair as final content.`,
		`Copying this skeleton unchanged is invalid and will be rejected as scaffold-only output.`,
		`Exact authored document target: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md".`,
		`Exact manifest target: "/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json".`,
		`"path": "payments-overview.md"`,
		`"artifact_root": "reports/taskruns/run-1/staging/shards/payments"`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	for _, forbidden := range []string{
		`FIRST COLLECT ARTIFACT PAIR COMMAND:`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/payments-overview.md' <<'ACP_COLLECT_DOC'`,
		`cat > '/tmp/workspace/reports/taskruns/run-1/staging/shards/payments/shard-pack-manifest.json' <<'ACP_MANIFEST_JSON'`,
		`ACP_COLLECT_BOOTSTRAP_REPLACE_BEFORE_EXIT`,
		`unchanged bootstrap pair`,
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("normal collect first-action must not contain bootstrap marker/wording %q:\n%s", forbidden, section)
		}
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
		`Produce one concise evidence-backed root overview document in write_root, then write an enriched shard-pack-manifest.json`,
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
	if got, want := manifest.Documents[0].CanonicalPath, "reports/as-is/bank-source-docs/docs-overview.md"; got != want {
		t.Fatalf("documents[0].canonical_path = %q, want %q", got, want)
	}
	if got, want := manifest.Documents[1].CanonicalPath, "reports/as-is/bank-source-docs/overview.md"; got != want {
		t.Fatalf("documents[1].canonical_path = %q, want %q", got, want)
	}
	if got, want := manifest.Citations[0].Path, "src/main.go"; got != want {
		t.Fatalf("citation path = %q, want %q", got, want)
	}
	if got := len(manifest.Semantic.Entities); got < 2 {
		t.Fatalf("semantic entities length = %d, want >= 2 in skeleton:\n%s", got, raw)
	}
	if got := len(manifest.Semantic.Edges); got < 1 {
		t.Fatalf("semantic edges length = %d, want >= 1 in skeleton:\n%s", got, raw)
	}
	if got := len(manifest.Semantic.Findings); got < 1 {
		t.Fatalf("semantic findings length = %d, want >= 1 in skeleton:\n%s", got, raw)
	}
	for _, entity := range manifest.Semantic.Entities {
		if len(entity.Provenance.Evidence) == 0 {
			t.Fatalf("semantic entity %s has no provenance evidence in skeleton:\n%s", entity.ID, raw)
		}
		if got, want := entity.Provenance.Evidence[0].Path, "src/main.go"; got != want {
			t.Fatalf("semantic entity %s evidence path = %q, want %q", entity.ID, got, want)
		}
	}
	if got, want := manifest.Semantic.Findings[0].Provenance.Evidence[0].Path, "src/main.go"; got != want {
		t.Fatalf("semantic finding evidence path = %q, want %q", got, want)
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
	if got := len(manifest.Semantic.Entities); got < 2 {
		t.Fatalf("refresh semantic entities length = %d, want >= 2 in skeleton:\n%s", got, raw)
	}
	if got := len(manifest.Semantic.Edges); got < 1 {
		t.Fatalf("refresh semantic edges length = %d, want >= 1 in skeleton:\n%s", got, raw)
	}
	if got := len(manifest.Semantic.Findings); got < 1 {
		t.Fatalf("refresh semantic findings length = %d, want >= 1 in skeleton:\n%s", got, raw)
	}
	question := manifest.Semantic.Questions[0]
	if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.Text) == "" {
		t.Fatalf("refresh skeleton question must include id and text, got %+v in:\n%s", question, raw)
	}
	if len(question.RelatedIDs) == 0 {
		t.Fatalf("refresh skeleton question must reference the scoped semantic entity, got %+v in:\n%s", question, raw)
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
	if got, want := manifest.Semantic.Entities[0].Provenance.Evidence[0].Path, "README.md"; got != want {
		t.Fatalf("root-file semantic evidence path = %q, want %q", got, want)
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
		`Same provider turn requirement`,
		`fresh-overwrite charter-overview.md before final success`,
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
	charter := runtimeDraftFirstActionFileTemplate(task, runtimedrafts.Output{
		Path:          "charter-overview.md",
		CanonicalPath: "charter/overview.md",
		Kind:          "charter",
		Title:         "Constitution",
	})
	for _, forbidden := range []string{
		"collected shard evidence",
		"validator output",
		"final index",
		"runtime repair",
		"downstream pipeline",
	} {
		if strings.Contains(strings.ToLower(charter), forbidden) {
			t.Fatalf("constitution bootstrap scaffold must not contain downstream/runtime-only wording %q:\n%s", forbidden, charter)
		}
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

func TestAsIsFirstActionSectionWritesEvidenceBackedDraftSetFirst(t *testing.T) {
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
		`one bounded evidence-read/write filesystem work unit`,
		`read current-run staged evidence first`,
		`then write asis-draft-manifest.json last before returning`,
		`a manifest-only first write before markdown is invalid`,
		`Same provider turn requirement`,
		`validation-ready markdown files exist`,
		`Your first response item for this step must be the filesystem command itself`,
		`Exact as-is draft manifest target: "/tmp/workspace/reports/taskruns/run-1/asis/asis-draft-manifest.json"`,
		`Exact overview target: "/tmp/workspace/reports/taskruns/run-1/staging/final/overview.md"`,
		`Exact coverage summary target: "/tmp/workspace/reports/taskruns/run-1/staging/final/summary.md"`,
		`Exact architect summary target: "/tmp/workspace/reports/taskruns/run-1/staging/final/architect-summary.md"`,
		`Run one filesystem command as the next action`,
		`the first provider item must be command_execution`,
		`perform only a bounded current-run evidence read/list`,
		`write all three markdown targets first, then write the manifest last before returning`,
		`AS-IS FIRST-PASS WRITE SEQUENCE:`,
		`write_root='/tmp/workspace/reports/taskruns/run-1/asis'`,
		`draft_root='/tmp/workspace/reports/taskruns/run-1/staging/final'`,
		`mkdir -p "$write_root" "$draft_root"`,
		`bounded evidence reads/lists from the current-run staged evidence index below`,
		`Markdown writes must preserve literal backticks and paths`,
		`single-quoted heredocs such as <<'EOF' or a python3 - <<'PY' program`,
		`Do not put markdown content that contains backticks inside double-quoted shell strings or unquoted heredocs`,
		`write evidence-backed markdown to "$draft_root/overview.md", "$draft_root/summary.md", and "$draft_root/architect-summary.md"`,
		`write "$write_root/asis-draft-manifest.json" last`,
		`test -s "$draft_root/overview.md"`,
		`test -s "$draft_root/summary.md"`,
		`test -s "$draft_root/architect-summary.md"`,
		`test -s "$write_root/asis-draft-manifest.json"`,
		`do not rely on focused repair to create it later`,
		`no empty evidence slots such as "from  and", "checked:  and", "under .", or "Use  and"`,
		`Architecture Home must never reference reports/taskruns/**, taskrun staging paths, write_root, draft_final_root, raw runtime artifacts, absolute runtime checkout paths, or .acp/repos paths`,
		`Every repo:path reference in Architecture Home must name an existing file or directory under that repository's current read root`,
		`Architecture Home must not use current run/current-run, typed shard, shard pack/manifest, or planned/succeeded/failed/incomplete counter wording`,
		`those execution details belong only in summary.md and architect-summary.md`,
		`AS-IS DRAFT MANIFEST SHAPE GUIDE:`,
		`omit downstream index availability entirely`,
		`"run_id": "run-1"`,
		`"step_id": "init.step2.asis_docs"`,
		`"step_contract": "as_is"`,
		`"summary": "Evidence-backed manifest for provider-authored runtime artifacts."`,
		`"canonical_path": "reports/as-is/overview.md"`,
		`"canonical_path": "reports/coverage/summary.md"`,
		`"canonical_path": "reports/agent-outputs/architect/summary.md"`,
		`Current-run staged evidence index`,
		`AS-IS FIRST-PASS SELF-CHECK:`,
		`The first write set was not manifest-only`,
		`summary.md and architect-summary.md contain the exact planned=<n> succeeded=<n> failed=<n> incomplete=<n> literal`,
		`explicit no-shard-coverage-blocker statement that current-run shard coverage is not a blocker`,
		`architect-summary.md says what is complete, what is missing, what the operator should inspect or decide next, and residual risk`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected as-is first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST AS-IS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected as-is first-action command once, got %d:\n%s", got, section)
	}
	for _, forbidden := range []string{
		"cat >",
		"ACP_DRAFT_MANIFEST_JSON",
		"ACP_DRAFT_FILE",
		"Provider wrote this draft artifact",
		"Provider wrote the required",
		"Drafted required runtime artifacts",
		"drafted required runtime artifacts",
		"The first draft artifact set is bootstrap-only",
		"no findings reported.",
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("as-is first-action drafts must not include placeholder marker %q:\n%s", forbidden, section)
		}
	}
}

func TestDocFirstFilesystemPolicyIncludesCurrentRunEvidenceIndex(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runID := "run-1"
	finalRoot := filepath.Join(workspace, "reports", "taskruns", runID, "staging", "final")
	emptyJSONPaths := []string{
		filepath.Join(finalRoot, "final-run-index.json"),
		filepath.Join(finalRoot, "citation-index.json"),
		filepath.Join(finalRoot, "reports", "coverage", "summary.md"),
		filepath.Join(workspace, "reports", "taskruns", runID, "staging", "shards", "bank", "shard-pack-manifest.json"),
	}
	for _, path := range emptyJSONPaths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir fixture path: %v", err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write fixture path: %v", err)
		}
	}
	findingsPath := filepath.Join(finalRoot, "reports", "findings", "findings.md")
	if err := os.MkdirAll(filepath.Dir(findingsPath), 0o755); err != nil {
		t.Fatalf("mkdir findings path: %v", err)
	}
	if err := os.WriteFile(findingsPath, []byte("# Findings\n\n- ID: `finding.bank.owner.gap`\n- Severity: `medium`\n"), 0o644); err != nil {
		t.Fatalf("write findings fixture: %v", err)
	}
	shardSummaryPath := filepath.Join(workspace, "reports", "taskruns", runID+"-init-step1-collect-shard-summary-bank.json")
	if err := os.MkdirAll(filepath.Dir(shardSummaryPath), 0o755); err != nil {
		t.Fatalf("mkdir shard summary path: %v", err)
	}
	if err := os.WriteFile(shardSummaryPath, []byte(`{"items":[{"status":"succeeded"},{"status":"succeeded"},{"status":"failed"},{"status":"pending"}]}`+"\n"), 0o644); err != nil {
		t.Fatalf("write shard summary fixture: %v", err)
	}
	task := acpruntime.Task{
		RunID:            runID,
		StepID:           "init.step2.asis_docs",
		StepContract:     "as_is",
		Workspace:        workspace,
		WriteRoot:        filepath.Join(workspace, "reports", "taskruns", runID, "asis"),
		DraftFinalRoot:   finalRoot,
		ReadContextRoots: []string{finalRoot},
		AgentRole:        "architect",
	}

	policy := DocFirstFilesystemPolicy(task)
	for _, needle := range []string{
		"Current-run staged evidence index",
		filepath.ToSlash(filepath.Join(finalRoot, "final-run-index.json")),
		filepath.ToSlash(filepath.Join(finalRoot, "citation-index.json")),
		filepath.ToSlash(filepath.Join(finalRoot, "reports", "coverage", "summary.md")),
		filepath.ToSlash(filepath.Join(workspace, "reports", "taskruns", runID+"-init-step1-collect-shard-summary-bank.json")),
		"planned=<n> succeeded=<n> failed=<n> incomplete=<n>",
		"planned=4 succeeded=2 failed=1 incomplete=1",
		"summary.md and architect-summary.md must state exact shard completeness counts",
		"explicit no-shard-coverage-blocker statement",
	} {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected as-is policy to contain %q, got:\n%s", needle, policy)
		}
	}

	task.StepID = "init.step4.proposals"
	task.StepContract = "proposals"
	task.WriteRoot = filepath.Join(workspace, "reports", "taskruns", runID, "proposals")
	policy = DocFirstFilesystemPolicy(task)
	for _, needle := range []string{
		"Exact current-run findings source",
		"reports/taskruns/run-1/staging/final/reports/findings/findings.md",
		filepath.ToSlash(filepath.Join(finalRoot, "reports", "findings", "findings.md")),
		"Do not use synthetic finding placeholders",
		"High/medium findings require one bullet per finding",
		"finding.bank.owner.gap severity=medium",
	} {
		if !strings.Contains(policy, needle) {
			t.Fatalf("expected proposals policy to contain %q, got:\n%s", needle, policy)
		}
	}
}

func TestProposalsFirstActionSectionWritesEvidenceBackedDraftSetFirst(t *testing.T) {
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
		`one bounded evidence-read/write filesystem work unit`,
		`read current-run staged findings/coverage/index evidence first`,
		`then write proposals-draft-manifest.json last before returning`,
		`a manifest-only first write before proposal/changelog markdown is invalid`,
		`Same provider turn requirement`,
		`validation-ready markdown files exist`,
		`proposal.md and changelog.md must both contain the exact literal shard completeness shape planned=<n> succeeded=<n> failed=<n> incomplete=<n>`,
		`both files must also state an explicit no-shard-coverage-blocker`,
		`staging/final/reports/*`,
		`Treat every reports/taskruns/** or staging/** path as an input locator only`,
		`Do not copy those paths into proposal.md or changelog.md`,
		`bullet-only Top Actionable Findings section`,
		`do not write Finding ID: none, Finding ID: n/a, Finding ID: unavailable`,
		`proposal.md must contain non-empty sections named Decision / recommended operator action, Evidence used, Proposed changes or follow-up plan, and Risks, gaps, and out-of-scope notes`,
		`changelog.md must contain non-empty sections named Updated architecture/proposal surfaces, Findings/proposals summary, Evidence index or citation references, and Residual coverage gaps`,
		`Do not use markdown tables for actionable findings`,
		`no structured current-run finding ID`,
		`finding unavailable`,
		`Every medium/high finding represented in the first-pass proposal uses one same-line bullet`,
		`Exact proposals draft manifest target: "/tmp/workspace/reports/taskruns/run-1/proposals/proposals-draft-manifest.json"`,
		`Exact proposal target: "/tmp/workspace/reports/taskruns/run-1/staging/final/proposal.md"`,
		`Exact changelog target: "/tmp/workspace/reports/taskruns/run-1/staging/final/changelog.md"`,
		`Run one filesystem command as the next action`,
		`perform only a bounded current-run evidence read/list`,
		`write proposal.md and changelog.md first with all required sections, then write the manifest last before returning`,
		`PROPOSALS DRAFT MANIFEST SHAPE GUIDE:`,
		`"run_id": "run-1"`,
		`"step_id": "init.step4.proposals"`,
		`"step_contract": "proposals"`,
		`"summary": "Evidence-backed manifest for provider-authored runtime artifacts."`,
		`"canonical_path": "proposals/runtime-recommendations.md"`,
		`"canonical_path": "reports/changelog/runtime-proposals.md"`,
		`Current-run staged evidence index`,
		`proposal.md and changelog.md must both state exact shard completeness in this literal shape`,
		`PROPOSALS FIRST-PASS SELF-CHECK:`,
		`No markdown target contains reports/taskruns/**, staging/final/**, staging/shards/**`,
		`The first write set was not manifest-only`,
		`proposal.md contains Decision / recommended operator action, Evidence used, Proposed changes or follow-up plan, and Risks, gaps, and out-of-scope notes`,
		`changelog.md contains Updated architecture/proposal surfaces, Findings/proposals summary, Evidence index or citation references, and Residual coverage gaps`,
		`proposal.md and changelog.md both include the exact planned=<n> succeeded=<n> failed=<n> incomplete=<n> literal`,
		`every Finding ID field uses an exact current-run finding ID, never none/n/a/unavailable`,
	}
	for _, needle := range required {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected proposals first-action section to contain %q, got:\n%s", needle, section)
		}
	}
	if got := strings.Count(section, "FIRST PROPOSALS DRAFT COMMAND:"); got != 1 {
		t.Fatalf("expected proposals first-action command once, got %d:\n%s", got, section)
	}
	for _, forbidden := range []string{
		"cat >",
		"ACP_DRAFT_MANIFEST_JSON",
		"ACP_DRAFT_FILE",
		"Provider wrote this draft artifact",
		"Provider wrote the required",
		"Drafted required runtime artifacts",
		"drafted required runtime artifacts",
		"The first proposals draft artifact set is bootstrap-only",
		"no findings reported.",
	} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("proposals first-action drafts must not include placeholder marker %q:\n%s", forbidden, section)
		}
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
		`Use the FIRST AS-IS DRAFT COMMAND above as an evidence-first write contract`,
		`The first draft artifact set must already be validation-ready`,
		`read bounded staged evidence first`,
		`first-pass content is evidence-backed as-is content`,
		`Architecture Home must never reference reports/taskruns/**, taskrun staging paths, write_root, draft_final_root, raw runtime artifacts, absolute runtime checkout paths, or .acp/repos paths`,
		`Every repo:path reference in Architecture Home must name an existing file or directory under that repository's current read root`,
		`Architecture Home must not use current run/current-run, typed shard, shard pack/manifest, or planned/succeeded/failed/incomplete counter wording`,
		`those execution details belong only in summary.md and architect-summary.md`,
		`Use staged final evidence from read_context_roots only; do NOT read sibling baseline workspaces`,
		`asis-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="as_is", agent_role, and outputs[]; optional top-level metadata is limited to summary and updated_at.`,
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
		`Use the FIRST PROPOSALS DRAFT COMMAND above as an evidence-first write contract`,
		`The first proposals draft artifact set must already be validation-ready`,
		`read current-run findings/coverage/index evidence first`,
		`first-pass content is evidence-backed proposal/changelog content`,
		`proposals-draft-manifest.json MUST include version=1, run_id, step_id, step_contract="proposals", agent_role, outputs[], and optional summary/updated_at.`,
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

func TestCollectPathScopeFileHintsKeepEachDirectoryScope(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, file := range []string{
		"cli/README.md",
		"cli/Cargo.toml",
		"cli/src/main.rs",
		"common/README.md",
		"common/hogql_parser/pyproject.toml",
		"common/hogvm/README.md",
	} {
		path := filepath.Join(repoRoot, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte(file+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	hints := CollectPathScopeFileHints(acpruntime.Task{
		StepID:           "init.step1.collect",
		ReadContextRoots: []string{repoRoot},
		PathScopes:       []string{"cli", "common"},
	})
	joined := strings.Join(hints, "\n")
	for _, needle := range []string{"cli/README.md", "cli/Cargo.toml", "common/README.md", "common/hogql_parser/pyproject.toml"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected path-scope hints to contain %q, got:\n%s", needle, joined)
		}
	}
}

func TestCollectFirstActionSectionIncludesPathScopeFileHints(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, file := range []string{"cli/README.md", "common/README.md"} {
		path := filepath.Join(repoRoot, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte(file+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	section := CollectFirstActionSection(acpruntime.Task{
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		WriteRoot:        "/tmp/workspace/reports/taskruns/run-1/staging/shards/cli-common",
		ReadContextRoots: []string{repoRoot},
		ShardID:          "cli-common",
		DomainID:         "posthog",
		RepoScopes:       []string{"posthog"},
		PathScopes:       []string{"cli", "common"},
	})
	for _, needle := range []string{
		"Existing path-scope file candidates for this collect shard:",
		"cli/README.md",
		"common/README.md",
	} {
		if !strings.Contains(section, needle) {
			t.Fatalf("expected collect first action section to contain %q, got:\n%s", needle, section)
		}
	}
}
