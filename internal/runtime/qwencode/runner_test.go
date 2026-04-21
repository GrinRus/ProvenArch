package qwencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func loadCapturedLiveStdoutFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "testdata", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live stdout fixture: %v", err)
	}
	return string(raw)
}

func loadRuntimeFixture(t *testing.T, name string) string {
	t.Helper()
	return loadCapturedLiveStdoutFixture(t, name)
}

func setCollectStallTimingsForTest(t *testing.T, window time.Duration, poll time.Duration, grace time.Duration) {
	t.Helper()
	prevPostWindow := collectPostArtifactStallWindow
	prevPoll := collectStallPollInterval
	prevGrace := collectStallTerminateGrace
	collectPostArtifactStallWindow = window
	collectStallPollInterval = poll
	collectStallTerminateGrace = grace
	t.Cleanup(func() {
		collectPostArtifactStallWindow = prevPostWindow
		collectStallPollInterval = prevPoll
		collectStallTerminateGrace = prevGrace
	})
}

func setCollectPreArtifactStallTimingsForTest(t *testing.T, window time.Duration, poll time.Duration, grace time.Duration) {
	t.Helper()
	prevWindow := collectPreArtifactStallWindow
	prevPoll := collectStallPollInterval
	prevGrace := collectStallTerminateGrace
	collectPreArtifactStallWindow = window
	collectStallPollInterval = poll
	collectStallTerminateGrace = grace
	t.Cleanup(func() {
		collectPreArtifactStallWindow = prevWindow
		collectStallPollInterval = prevPoll
		collectStallTerminateGrace = prevGrace
	})
}

func validRetryRichManifestJSON() string {
	return `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["iac"],
  "summary": "Collected repo-specific infrastructure evidence.",
  "documents": [
    {
      "id": "doc.iac.overview",
      "kind": "report",
      "title": "IAC Overview",
      "path": "iac-overview.md",
      "canonical_path": "reports/as-is/iac-overview.md",
      "topics": ["iac"],
      "citation_ids": ["cite.bank.readme", "cite.bank.kustomization"],
      "status": "staged"
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "iac/README.md",
      "claim_ids": ["claim.iac.readme"],
      "document_ids": ["doc.iac.overview"]
    },
    {
      "id": "cite.bank.kustomization",
      "repo": "bank-of-anthos",
      "path": "iac/acm-multienv-cicd-anthos-autopilot/base/kustomization.yaml",
      "claim_ids": ["claim.iac.kustomization"],
      "document_ids": ["doc.iac.overview"]
    }
  ],
  "compatibility": {
    "coverage": {
      "observed": ["iac"],
      "missing": ["owner mappings"],
      "notes": ["repo evidence preserved"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func validConstitutionDraftManifestJSON(runID string) string {
	if strings.TrimSpace(runID) == "" {
		runID = "run-1"
	}
	return fmt.Sprintf(`{
  "version": 1,
  "run_id": %q,
  "step_id": "init.step0.constitution",
  "step_contract": "constitution",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "charter-overview.md",
      "canonical_path": "charter/overview.md",
      "kind": "charter",
      "title": "Constitution"
    },
    {
      "path": "baseline-subagents.yaml",
      "canonical_path": "skills/subagents.yaml",
      "kind": "bundle",
      "title": "Baseline Subagents"
    }
  ]
}`, runID)
}

func validAsIsDraftManifestJSON(runID string) string {
	if strings.TrimSpace(runID) == "" {
		runID = "run-1"
	}
	return fmt.Sprintf(`{
  "version": 1,
  "run_id": %q,
  "step_id": "refresh.step2.asis_docs",
  "step_contract": "as_is",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "bank-overview.md",
      "canonical_path": "reports/as-is/bank-overview.md",
      "kind": "report",
      "title": "Bank Overview"
    }
  ]
}`, runID)
}

func validProposalsDraftManifestJSON(runID string) string {
	if strings.TrimSpace(runID) == "" {
		runID = "run-1"
	}
	return fmt.Sprintf(`{
  "version": 1,
  "run_id": %q,
  "step_id": "refresh.step4.proposals",
  "step_contract": "proposals",
  "agent_role": "architect",
  "outputs": [
    {
      "path": "proposal.md",
      "canonical_path": "proposals/proposal-baseline/proposal.md",
      "kind": "proposal",
      "title": "Proposal"
    }
  ]
}`, runID)
}

func validRetrySkeletalManifestJSON() string {
	return `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-docs",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["docs"],
  "documents": [
    {
      "id": "doc.reused.analysis",
      "kind": "report",
      "title": "Reused Analysis",
      "path": "shard-analysis.md",
      "canonical_path": "reports/as-is/shard-analysis.md",
      "topics": ["docs"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.reused.analysis"]
    }
  ],
  "summary": "Reused existing shard artifacts.",
  "compatibility": {
    "coverage": {
      "observed": ["docs"],
      "missing": ["owner mappings"],
      "notes": ["reused existing artifacts"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func validRetryUnlinkedRepoSpecificManifestJSON() string {
	return `{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-docs",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["docs"],
  "documents": [
    {
      "id": "doc.reused.analysis",
      "kind": "report",
      "path": "shard-analysis.md",
      "title": "Reused Analysis",
      "canonical_path": "reports/as-is/overview.md",
      "topics": ["docs"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.reused.analysis"]
    },
    {
      "id": "cite.bank.repo-root",
      "repo": "bank-of-anthos",
      "path": "docs/architecture.md",
      "claim_ids": ["claim.repo.root"],
      "document_ids": ["doc.reused.analysis"]
    }
  ],
  "summary": "Reused existing shard artifacts.",
  "compatibility": {
    "coverage": {
      "observed": ["docs"],
      "missing": ["owner mappings"],
      "notes": ["reused existing artifacts"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func validRetryRichManifestMissingMetadataJSON() string {
	return `{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs",
  "version": 1,
  "documents": [
    {
      "id": "doc.service-catalog",
      "kind": "analysis",
      "title": "Bank of Anthos Service Catalog",
      "path": "service-catalog.md",
      "canonical_path": "reports/as-is/bank-of-anthos/service-catalog.md",
      "topics": ["services", "architecture"],
      "citation_ids": ["cite.readme", "cite.pom"]
    }
  ],
  "citations": [
    {
      "id": "cite.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.services"],
      "document_ids": ["doc.service-catalog"]
    },
    {
      "id": "cite.pom",
      "repo": "bank-of-anthos",
      "path": "pom.xml",
      "claim_ids": ["claim.maven-modules"],
      "document_ids": ["doc.service-catalog"]
    }
  ]
}`
}

func invalidLegacyStyleManifestJSON() string {
	return `{
  "version": "1.0.0",
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-docs",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs",
  "repo_scope": "bank-of-anthos",
  "path_scopes": ["docs"],
  "documents": [
    {
      "path": "services.md",
      "canonical_path": "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs/services.md",
      "topics": ["services"],
      "citations": [
        { "repo": "bank-of-anthos", "path": "docs/development.md" }
      ]
    }
  ],
  "citations": [
    { "repo": "bank-of-anthos", "path": "docs/development.md" }
  ],
  "compatibility": {
    "coverage": {
      "observed": ["docs"],
      "missing": ["owner mappings"],
      "notes": ["legacy manifest drift"]
    },
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}`
}

func TestHeadlessRunnerUnavailableClassifiesAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{Command: "definitely-missing-acp-qwen-command"}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected headless runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "is unavailable") {
		t.Fatalf("unexpected error message %q", message)
	}
}

func TestHeadlessRunnerUnavailableIncludesStdoutExcerptWhenStderrEmpty(t *testing.T) {
	t.Parallel()

	commandPath := filepath.Join(t.TempDir(), "qwen-unavailable-stdout.sh")
	script := `#!/bin/sh
set -eu
echo "qwen failed due to transient setup issue"
exit 1
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write unavailable stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-unavailable-stdout",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "stdout_excerpt=") {
		t.Fatalf("expected stdout excerpt in unavailable error message, got %q", message)
	}
	if !strings.Contains(message, "parse_stage=exec") || !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw-output diagnostics in unavailable error message, got %q", message)
	}
}

func TestHeadlessRunnerUnsupportedPromptFlagClassifiesAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	commandPath := filepath.Join(t.TempDir(), "qwen-unsupported-prompt.sh")
	script := `#!/bin/sh
set -eu
echo "unknown option --prompt" >&2
exit 1
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write unsupported-prompt stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-unsupported-prompt",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner unavailable error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "runner is unavailable") {
		t.Fatalf("expected runner unavailable marker in error message, got %q", message)
	}
}

func TestHeadlessRunnerInvalidTaskResultClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			"echo '{\"response\":\"not-json\"}'",
		},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-2",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "invalid taskresult") || !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("unexpected error message %q", message)
	}
	if !strings.Contains(message, "parse_stage=") {
		t.Fatalf("expected parse_stage marker in parse-fail message, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected parse-fail raw output metadata in %s", filepath.Join(workspace, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerClassifiesOpenstackTransportTranscriptAsRunnerUnavailable(t *testing.T) {
	workspace := t.TempDir()
	commandPath := filepath.Join(workspace, "qwen-openstack-ssl-transport.sh")
	fixturePath := filepath.Join(workspace, "openstack-ssl.txt")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_openstack_step0_ssl_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write ssl fixture copy: %v", err)
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
cat %q
`, fixturePath)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write transport stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-openstack-step0",
		RunID:        "run-1",
		StepID:       "init.step0.constitution",
		Workspace:    workspace,
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner_unavailable for transport transcript")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("expected runner_unavailable, got %s", code)
	}
	if !strings.Contains(message, "parse_stage=transport") {
		t.Fatalf("expected transport parse stage in error, got %q", message)
	}
	if !strings.Contains(strings.ToLower(message), "ssl") {
		t.Fatalf("expected ssl marker in unavailable message, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output artifact path in message, got %q", message)
	}
}

func TestHeadlessRunnerClassifiesOpenstackPermissionErrorTranscriptAsRunnerUnavailable(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	commandPath := filepath.Join(workspace, "qwen-openstack-permission-error.sh")
	fixturePath := filepath.Join(workspace, "openstack-permission-error.txt")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_openstack_permission_error_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write permission fixture copy: %v", err)
	}
	script := fmt.Sprintf(`#!/bin/sh
set -eu
cat %q
`, fixturePath)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write permission-error stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-openstack-step3",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		Workspace:    workspace,
		RepoScopes:   []string{"openstack"},
		StartedAtUTC: time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner_unavailable for permission_error transcript")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerUnavailable) {
		t.Fatalf("expected runner_unavailable, got %s", code)
	}
	if !strings.Contains(message, "parse_stage=transport") {
		t.Fatalf("expected transport parse stage in error, got %q", message)
	}
	lower := strings.ToLower(message)
	if !strings.Contains(lower, "permission_error") && !strings.Contains(lower, "usage limit") {
		t.Fatalf("expected permission/quota marker in unavailable message, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output artifact path in message, got %q", message)
	}
}

func TestHeadlessRunnerRepairsInvalidStep2DraftManifestOnce(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	stateFile := filepath.Join(tempDir, "step2-draft-repair-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step2-draft-repair.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
mkdir -p %q %q
cat > %q/bank-overview.md <<'DOC'
# Bank Overview
DOC
if [ "$count" -eq 1 ]; then
  cat > %q/asis-draft-manifest.json <<'JSON'
{"version":"0.1.0","outputs":[]}
JSON
  cat <<'JSON'
{"meta":{"task_id":"task-step2-draft-repair","step_id":"refresh.step2.asis_docs","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"initial step2 as-is draft","changeset":[]}
JSON
  exit 0
fi
if ! echo "$last_arg" | grep -Fq 'asis-draft-manifest.json'; then
  echo "missing asis manifest retry hint" >&2
  exit 91
fi
if ! echo "$last_arg" | grep -Fq '"changeset": []'; then
  echo "missing empty changeset repair guidance" >&2
  exit 92
fi
cat > %q/asis-draft-manifest.json <<'JSON'
%s
JSON
cat <<'JSON'
{"meta":{"task_id":"task-step2-draft-repair","step_id":"refresh.step2.asis_docs","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"repaired step2 as-is draft","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, draftRoot, writeRoot, writeRoot, validAsIsDraftManifestJSON("run-1"))
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step2 repair stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step2-draft-repair",
		RunID:             "run-1",
		StepID:            "refresh.step2.asis_docs",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "as_is",
		ExpectedArtifacts: []string{"asis-draft-manifest.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected step2 draft repair to succeed: %v", err)
	}
	if result.TaskResult.Summary != "repaired step2 as-is draft" {
		t.Fatalf("unexpected repaired summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
}

func TestHeadlessRunnerFailsWhenStep4DraftRepairLeavesMissingReferencedFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	stateFile := filepath.Join(tempDir, "step4-draft-repair-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step4-draft-repair-fail.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
mkdir -p %q %q
if [ "$count" -eq 1 ]; then
  cat > %q/proposals-draft-manifest.json <<'JSON'
{"version":"0.1.0","outputs":[]}
JSON
  cat <<'JSON'
{"meta":{"task_id":"task-step4-draft-repair-fail","step_id":"refresh.step4.proposals","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"initial step4 proposals draft","changeset":[]}
JSON
  exit 0
fi
if ! echo "$last_arg" | grep -Fq 'proposals-draft-manifest.json'; then
  echo "missing proposals manifest retry hint" >&2
  exit 91
fi
if ! echo "$last_arg" | grep -Fq '"changeset": []'; then
  echo "missing empty changeset repair guidance" >&2
  exit 92
fi
cat > %q/proposals-draft-manifest.json <<'JSON'
%s
JSON
cat <<'JSON'
{"meta":{"task_id":"task-step4-draft-repair-fail","step_id":"refresh.step4.proposals","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"repair kept missing proposal draft file","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, writeRoot, writeRoot, validProposalsDraftManifestJSON("run-1"))
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step4 repair fail stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step4-draft-repair-fail",
		RunID:             "run-1",
		StepID:            "refresh.step4.proposals",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "proposals",
		ExpectedArtifacts: []string{"proposals-draft-manifest.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner_parse_failed when step4 repair leaves missing draft file")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if !strings.Contains(message, "runtime required draft artifacts remained invalid after one repair attempt") {
		t.Fatalf("expected draft artifact repair failure context, got %q", message)
	}
	if !strings.Contains(message, "referenced draft file") {
		t.Fatalf("expected explicit missing draft file reason, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output artifact path in message, got %q", message)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
}

func TestHeadlessRunnerRepairsInvalidStep0DraftManifestOnce(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	legacyFixturePath := filepath.Join(tempDir, "legacy-step0.json")
	if err := os.WriteFile(legacyFixturePath, []byte(loadRuntimeFixture(t, "qwen_step0_bank_single_legacy_constitution_draft.json")), 0o644); err != nil {
		t.Fatalf("write legacy fixture: %v", err)
	}

	stateFile := filepath.Join(tempDir, "step0-repair-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step0-repair.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
mkdir -p %q %q
cat > %q/charter-overview.md <<'DOC'
# Constitution
DOC
cat > %q/baseline-subagents.yaml <<'DOC'
version: 1
profiles:
  - id: baseline-architect
    role: architect
DOC
if [ "$count" -eq 1 ]; then
  cp %q %q/constitution-draft.json
  cat <<'JSON'
{"meta":{"task_id":"task-step0-repair","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"initial step0 constitution draft","changeset":[]}
JSON
  exit 0
fi
if ! echo "$last_arg" | grep -Fq 'constitution-draft.json'; then
  echo "missing constitution manifest retry hint" >&2
  exit 91
fi
if ! echo "$last_arg" | grep -Fq '"step_contract": "constitution"'; then
  echo "missing exact constitution manifest contract example" >&2
  exit 92
fi
if ! echo "$last_arg" | grep -Fq '"canonical_path": "charter/overview.md"'; then
  echo "missing charter canonical path hint" >&2
  exit 93
fi
if ! echo "$last_arg" | grep -Fq '"changeset": []'; then
  echo "missing empty changeset repair guidance" >&2
  exit 94
fi
cat > %q/constitution-draft.json <<'JSON'
%s
JSON
cat <<'JSON'
{"meta":{"task_id":"task-step0-repair","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"repaired step0 constitution draft","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, draftRoot, draftRoot, legacyFixturePath, writeRoot, writeRoot, validConstitutionDraftManifestJSON("run-1"))
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step0 repair stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step0-repair",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected step0 draft repair to succeed: %v", err)
	}
	if result.TaskResult.Summary != "repaired step0 constitution draft" {
		t.Fatalf("unexpected repaired summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	raw, readErr := os.ReadFile(filepath.Join(writeRoot, "constitution-draft.json"))
	if readErr != nil {
		t.Fatalf("read repaired constitution draft manifest: %v", readErr)
	}
	if !strings.Contains(string(raw), `"version": 1`) || !strings.Contains(string(raw), `"step_contract": "constitution"`) {
		t.Fatalf("expected repaired step0 manifest to stay canonical, got %q", string(raw))
	}
}

func TestHeadlessRunnerFailsWhenStep0DraftRepairKeepsLegacyManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	initialFixturePath := filepath.Join(tempDir, "legacy-step0-initial.json")
	if err := os.WriteFile(initialFixturePath, []byte(loadRuntimeFixture(t, "qwen_step0_bank_single_legacy_constitution_draft.json")), 0o644); err != nil {
		t.Fatalf("write initial legacy fixture: %v", err)
	}
	retryFixturePath := filepath.Join(tempDir, "legacy-step0-retry.json")
	if err := os.WriteFile(retryFixturePath, []byte(loadRuntimeFixture(t, "qwen_step0_openstack_multi_schema_version_constitution_draft.json")), 0o644); err != nil {
		t.Fatalf("write retry legacy fixture: %v", err)
	}

	stateFile := filepath.Join(tempDir, "step0-repair-fail-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step0-repair-fail.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q %q
cat > %q/charter-overview.md <<'DOC'
# Constitution
DOC
cat > %q/baseline-subagents.yaml <<'DOC'
version: 1
profiles:
  - id: baseline-architect
    role: architect
DOC
if [ "$count" -eq 1 ]; then
  cp %q %q/constitution-draft.json
  cat <<'JSON'
{"meta":{"task_id":"task-step0-repair-fail","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"initial step0 constitution draft","changeset":[]}
JSON
  exit 0
fi
cp %q %q/constitution-draft.json
cat <<'JSON'
{"meta":{"task_id":"task-step0-repair-fail","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"repair kept legacy constitution draft","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, draftRoot, draftRoot, initialFixturePath, writeRoot, retryFixturePath, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step0 repair fail stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step0-repair-fail",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"openstack"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected runner_parse_failed when step0 repair keeps legacy manifest")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if !strings.Contains(message, "runtime required draft artifacts remained invalid after one repair attempt") {
		t.Fatalf("expected draft artifact repair failure context, got %q", message)
	}
	if !strings.Contains(message, "runtime draft manifest version must be 1") {
		t.Fatalf("expected explicit draft manifest validation reason, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output artifact path in message, got %q", message)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(tempDir, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected raw output metadata under %s", filepath.Join(tempDir, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerClassifiesStep0DraftRepairExecFailureAsParseFailed(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}

	initialFixturePath := filepath.Join(tempDir, "legacy-step0-initial.json")
	if err := os.WriteFile(initialFixturePath, []byte(loadRuntimeFixture(t, "qwen_step0_bank_single_legacy_constitution_draft.json")), 0o644); err != nil {
		t.Fatalf("write initial legacy fixture: %v", err)
	}

	stateFile := filepath.Join(tempDir, "step0-repair-exec-fail-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step0-repair-exec-fail.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q %q
cat > %q/charter-overview.md <<'DOC'
# Constitution
DOC
cat > %q/baseline-subagents.yaml <<'DOC'
version: 1
profiles:
  - id: baseline-architect
    role: architect
DOC
if [ "$count" -eq 1 ]; then
  cp %q %q/constitution-draft.json
  cat <<'JSON'
{"meta":{"task_id":"task-step0-repair-exec-fail","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"initial step0 constitution draft","changeset":[]}
JSON
  exit 0
fi
echo "repair command crashed while rewriting constitution-draft.json" >&2
exit 17
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, draftRoot, draftRoot, initialFixturePath, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step0 repair exec-fail stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step0-repair-exec-fail",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse_failed when step0 repair command exits non-zero")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if strings.Contains(message, string(acpruntime.ErrorCodeRunnerUnavailable)) {
		t.Fatalf("did not expect runner_unavailable wording, got %q", message)
	}
	if !strings.Contains(message, "runtime required draft artifact repair attempt failed after initial validation error") {
		t.Fatalf("expected draft repair exec failure context, got %q", message)
	}
	if !strings.Contains(message, "runtime draft manifest version must be 1") {
		t.Fatalf("expected initial manifest validation reason, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output artifact path in message, got %q", message)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
}

func TestHeadlessRunnerAcceptsStep0DraftFilesWrittenAtCanonicalPaths(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "charter"), 0o755); err != nil {
		t.Fatalf("mkdir draft charter dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(draftRoot, "skills"), 0o755); err != nil {
		t.Fatalf("mkdir draft skills dir: %v", err)
	}

	commandPath := filepath.Join(tempDir, "qwen-step0-canonical-layout.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
mkdir -p %q %q/charter %q/skills
cat > %q/constitution-draft.json <<'JSON'
%s
JSON
cat > %q/charter/overview.md <<'DOC'
# Constitution
DOC
cat > %q/skills/subagents.yaml <<'DOC'
version: 1
profiles:
  - id: baseline-architect
    role: architect
DOC
cat <<'JSON'
{"meta":{"task_id":"task-step0-canonical-layout","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"step0 used canonical-path draft files","changeset":[]}
JSON
`, writeRoot, draftRoot, draftRoot, writeRoot, validConstitutionDraftManifestJSON("run-1"), draftRoot, draftRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step0 canonical-layout stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step0-canonical-layout",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected step0 canonical-path draft files to reconcile: %v", err)
	}
	if result.TaskResult.Summary != "step0 used canonical-path draft files" {
		t.Fatalf("unexpected summary %q", result.TaskResult.Summary)
	}
	if _, err := os.Stat(filepath.Join(draftRoot, "charter-overview.md")); err != nil {
		t.Fatalf("expected reconciled charter-overview.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(draftRoot, "baseline-subagents.yaml")); err != nil {
		t.Fatalf("expected reconciled baseline-subagents.yaml: %v", err)
	}
}

func TestHeadlessRunnerSchemaInvalidCandidateClassifiesAsSchemaStage(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"meta":{"task_id":"task-1"}}'`,
		},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "parse_stage=schema") {
		t.Fatalf("expected parse_stage=schema, got %q", message)
	}
}

func TestHeadlessRunnerRejectsDeprecatedEdgeAliasesOnSchemaValidation(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"meta":{"task_id":"task-edge-repair","step_id":"refresh.step3.findings","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[{"op":"upsert_edge","edge":{"id":"edge.a.b","kind":"depends_on","source":"svc.a","target":"svc.b","provenance":{"kind":"inference","confidence":0.7,"evidence":[{"repo":"repo-a","path":"README.md"}]}}}]}'`,
		},
	}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-edge-repair",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"repo-a", "repo-b"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error for deprecated edge aliases")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "parse_stage=schema") {
		t.Fatalf("expected parse_stage=schema, got %q", message)
	}
}

func TestHeadlessRunnerRetriesOnceOnParseFailureForDefaultArgs(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"meta":{"task_id":"task-1","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[{"op":"upsert_entity","entity":{"id":"svc.payments-service","type":"service","name":"Payments Service","attributes":{"repo_scope":"payments-service"},"provenance":{"kind":"observation","confidence":0.7,"evidence":[{"repo":"payments-service","path":"README.md"}]}}}]}
JSON
  exit 0
fi
echo '{"response":"not-json"}'
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write retry command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerDefaultArgsUsePromptFlagForExecution(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-prompt-flag-stub.sh")
	script := `#!/bin/sh
set -eu
has_prompt=0
prompt_value=""
expect_prompt_value=0
for arg in "$@"; do
  if [ "$expect_prompt_value" -eq 1 ]; then
    prompt_value="$arg"
    expect_prompt_value=0
    continue
  fi
  if [ "$arg" = "--prompt" ]; then
    has_prompt=1
    expect_prompt_value=1
    continue
  fi
done
if [ "$has_prompt" -ne 1 ] || [ -z "$prompt_value" ]; then
  echo "missing --prompt argument" >&2
  exit 1
fi
cat <<'JSON'
{"meta":{"task_id":"task-prompt-flag","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write prompt-flag command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-prompt-flag",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected default run to succeed with --prompt: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-prompt-flag" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerRetryParseFailureUsesRetryOutputInRunnerError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-retry-parse-fail-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  echo 'retry invalid payload'
  echo 'retry stderr detail' >&2
  exit 0
fi
echo 'first invalid payload'
echo 'first stderr detail' >&2
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write retry-parse-fail command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse-failed error")
	}

	code, _, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}

	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError details")
	}
	if !strings.Contains(runnerErr.Stdout, "retry invalid payload") {
		t.Fatalf("expected retry stdout in runner error, got %q", runnerErr.Stdout)
	}
	if strings.Contains(runnerErr.Stdout, "first invalid payload") {
		t.Fatalf("did not expect first-attempt stdout in runner error, got %q", runnerErr.Stdout)
	}
	if !strings.Contains(runnerErr.Stderr, "retry stderr detail") {
		t.Fatalf("expected retry stderr in runner error, got %q", runnerErr.Stderr)
	}
	if strings.Contains(runnerErr.Stderr, "first stderr detail") {
		t.Fatalf("did not expect first-attempt stderr in runner error, got %q", runnerErr.Stderr)
	}
}

func TestHeadlessRunnerRetriesCapturedLiveInvalidStdoutFixture(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-live-retry-stub.sh")
	fixturePath := filepath.Join(tempDir, "live-invalid-output.txt")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_bank_of_anthos_invalid_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write live fixture copy: %v", err)
	}
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY MODE"; then
  cat <<'JSON'
{"meta":{"task_id":"task-live-retry","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
  exit 0
fi
cat "$QWEN_LIVE_FIXTURE"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write live retry command: %v", err)
	}

	t.Setenv("QWEN_LIVE_FIXTURE", fixturePath)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-live-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed after captured live invalid stdout: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-live-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerRetriesCapturedLiveIACInvalidStdoutFixtureWithWriteRootRecoveryPrompt(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-live-iac-retry-stub.sh")
	fixturePath := filepath.Join(tempDir, "live-invalid-iac-output.txt")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_bank_of_anthos_iac_invalid_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write live iac fixture copy: %v", err)
	}
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "iac-overview.md"), []byte("# Overview\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "Retry goal is JSON repair, not fresh repository exploration." \
  && echo "$last_arg" | grep -q "shard-pack-manifest.json is already present in write_root"; then
  cat <<'JSON'
{"meta":{"task_id":"task-live-iac-retry","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
  exit 0
fi
cat "$QWEN_LIVE_IAC_FIXTURE"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write live iac retry command: %v", err)
	}

	t.Setenv("QWEN_LIVE_IAC_FIXTURE", fixturePath)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-live-iac-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed after captured live iac invalid stdout: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-live-iac-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerRetriesCapturedLiveExtrasInvalidStdoutFixtureWithMinimalRetryPrompt(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-live-extras-retry-stub.sh")
	fixturePath := filepath.Join(tempDir, "live-invalid-extras-output.txt")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_bank_of_anthos_extras_invalid_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write live extras fixture copy: %v", err)
	}
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-analysis.md"), []byte("# Analysis\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q 'Prefer "changeset": \[\]' \
  && echo "$last_arg" | grep -q "Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON." \
  && echo "$last_arg" | grep -q "Retry-safe minimal template (preferred when reusing existing write_root artifacts):"; then
  cat <<'JSON'
{"meta":{"task_id":"task-live-extras-retry","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}
JSON
  exit 0
fi
cat "$QWEN_LIVE_EXTRAS_FIXTURE"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write live extras retry command: %v", err)
	}

	t.Setenv("QWEN_LIVE_EXTRAS_FIXTURE", fixturePath)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-live-extras-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed after captured live extras invalid stdout: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-live-extras-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerRecoversCollectStallAfterArtifactsViaRetry(t *testing.T) {
	setCollectStallTimingsForTest(t, 150*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-stall-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["iac"],
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.iac.readme"],
      "document_ids": ["doc.iac"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["iac"], "missing": ["owner mappings"], "notes": ["recovered"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
  cat <<'JSON'
{"meta":{"task_id":"task-stall-retry","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"recovered","changeset":[]}
JSON
  exit 0
fi
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "version": 1,
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.iac"]
    }
  ]
}
JSON
printf 'artifacts-ready\n'
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write stall retry stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-stall-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected collect stall recovery to succeed: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-stall-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
	if len(diagnostics) < 3 {
		t.Fatalf("expected stall diagnostics, got %d events", len(diagnostics))
	}
	if diagnostics[0].Message != "runtime task stalled after artifacts" {
		t.Fatalf("expected first diagnostic to report stall, got %q", diagnostics[0].Message)
	}
	if diagnostics[1].Message != "runtime task retry scheduled" {
		t.Fatalf("expected retry scheduled diagnostic, got %q", diagnostics[1].Message)
	}
	if diagnostics[len(diagnostics)-1].Message != "runtime task retry completed" {
		t.Fatalf("expected retry completed diagnostic, got %q", diagnostics[len(diagnostics)-1].Message)
	}
	if diagnostics[len(diagnostics)-1].Fields["retry_status"] != "succeeded" {
		t.Fatalf("expected retry_status=succeeded, got %#v", diagnostics[len(diagnostics)-1].Fields["retry_status"])
	}
}

func TestHeadlessRunnerDoesNotTriggerCollectStallMonitorWhileActivityContinues(t *testing.T) {
	setCollectStallTimingsForTest(t, 160*time.Millisecond, 20*time.Millisecond, 10*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-stall-active-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  echo "retry-unexpected" >&2
  exit 97
fi
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["iac"],
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.iac.readme"],
      "document_ids": ["doc.iac"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["iac"], "missing": ["owner mappings"], "notes": ["active"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
JSON
for tick in 1 2 3 4; do
  printf 'tick-%s\n' "$tick"
  printf '\nupdate %s\n' "$tick" >> "$STALL_WRITE_ROOT/iac-analysis.md"
  sleep 0.05
done
cat <<'JSON'
{"meta":{"task_id":"task-stall-active","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"completed with live activity","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write active stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-stall-active",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected active collect run to finish without stall: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-stall-active" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
	for _, event := range diagnostics {
		if event.Message == "runtime task stalled after artifacts" {
			t.Fatalf("did not expect stall diagnostic while activity continued")
		}
	}
}

func TestHeadlessRunnerRecoversCollectStallBeforeArtifactsViaRetry(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, time.Second, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE" \
  && echo "$last_arg" | grep -q "no durable collect artifacts in time"; then
  mkdir -p "$STALL_WRITE_ROOT"
  printf '# Extras\n' > "$STALL_WRITE_ROOT/extras-analysis.md"
  cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["recovered"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
  cat <<'JSON'
{"meta":{"task_id":"task-pre-stall-retry","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"recovered after pre-artifact stall","changeset":[]}
JSON
  exit 0
fi
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre-stall retry stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-pre-stall-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected pre-artifact stall recovery to succeed: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-pre-stall-retry" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
	if len(diagnostics) < 3 {
		t.Fatalf("expected pre-artifact stall diagnostics, got %d events", len(diagnostics))
	}
	if diagnostics[0].Message != "runtime task stalled before artifacts" {
		t.Fatalf("expected first diagnostic to report pre-artifact stall, got %q", diagnostics[0].Message)
	}
	if diagnostics[1].Message != "runtime task retry scheduled" {
		t.Fatalf("expected retry scheduled diagnostic, got %q", diagnostics[1].Message)
	}
	if diagnostics[0].Fields["stall_phase"] != "pre_artifact" {
		t.Fatalf("expected stall_phase=pre_artifact, got %#v", diagnostics[0].Fields["stall_phase"])
	}
	if diagnostics[len(diagnostics)-1].Message != "runtime task retry completed" {
		t.Fatalf("expected retry completed diagnostic, got %q", diagnostics[len(diagnostics)-1].Message)
	}
	if diagnostics[len(diagnostics)-1].Fields["retry_status"] != "succeeded" {
		t.Fatalf("expected retry_status=succeeded, got %#v", diagnostics[len(diagnostics)-1].Fields["retry_status"])
	}
}

func TestHeadlessRunnerPreArtifactRecoveryLogsFinalPostArtifactRetryContext(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, time.Second, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	stateFile := filepath.Join(tempDir, "qwen-pre-stall-post-artifact-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-post-artifact.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if [ "$count" -eq 1 ]; then
  sleep 10
  exit 0
fi
if [ "$count" -eq 2 ]; then
  if ! echo "$last_arg" | grep -q 'no durable collect artifacts in time'; then
    echo "missing pre-artifact retry hint" >&2
    exit 91
  fi
  mkdir -p %q
  printf '# Extras\n' > %q/extras-analysis.md
  cat > %q/shard-pack-manifest.json <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["invalid findings form"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": ["finding string is invalid"]
  }
}
MANIFEST
  cat <<'JSON'
{"meta":{"task_id":"task-pre-post-context","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"fresh retry returned invalid manifest","changeset":[]}
JSON
  exit 0
fi
if [ "$count" -eq 3 ]; then
  if ! echo "$last_arg" | grep -Fq 'documents[].path MUST stay relative to artifact_root only'; then
    echo "missing artifact-root repair guidance" >&2
    exit 92
  fi
  cat > %q/shard-pack-manifest.json <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["recovered"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
  cat <<'JSON'
{"meta":{"task_id":"task-pre-post-context","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"recovered after pre-artifact then post-artifact repair","changeset":[]}
JSON
  exit 0
fi
echo "unexpected invocation $count" >&2
exit 93
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, writeRoot, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre/post context stub: %v", err)
	}

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-pre-post-context",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected combined pre/post recovery to succeed: %v", err)
	}
	if result.TaskResult.Summary != "recovered after pre-artifact then post-artifact repair" {
		t.Fatalf("unexpected result summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "3" {
		t.Fatalf("expected exactly 3 runner invocations, got %q", count)
	}
	last := diagnostics[len(diagnostics)-1]
	if last.Message != "runtime task retry completed" {
		t.Fatalf("expected retry completed diagnostic, got %q", last.Message)
	}
	if last.Fields["retry_status"] != "succeeded" {
		t.Fatalf("expected retry_status=succeeded, got %#v", last.Fields["retry_status"])
	}
	if last.Fields["last_stall_phase"] != "pre_artifact" {
		t.Fatalf("expected last_stall_phase=pre_artifact when follow-up repair did not come from a real second stall, got %#v", last.Fields["last_stall_phase"])
	}
	if last.Fields["manifest_state_before_retry"] != "invalid" {
		t.Fatalf("expected manifest_state_before_retry=invalid, got %#v", last.Fields["manifest_state_before_retry"])
	}
}

func TestHeadlessRunnerPreArtifactThenPostArtifactStallLogsFinalPostArtifactContext(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, 900*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)
	setCollectStallTimingsForTest(t, 250*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	stateFile := filepath.Join(tempDir, "qwen-pre-post-stall-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-pre-post-stall.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if [ "$count" -eq 1 ]; then
  sleep 10
  exit 0
fi
if [ "$count" -eq 2 ]; then
  if ! echo "$last_arg" | grep -q 'no durable collect artifacts in time'; then
    echo "missing pre-artifact retry hint" >&2
    exit 91
  fi
  mkdir -p %q
  printf '# Extras\n' > %q/extras-analysis.md
  cat > %q/shard-pack-manifest.json <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["stalled after artifacts"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
  printf 'artifacts-ready\n'
  (trap '' TERM INT HUP; sleep 3) &
  sleep 3
  exit 0
fi
if [ "$count" -eq 3 ]; then
  if ! echo "$last_arg" | grep -Fq 'documents[].path MUST stay relative to artifact_root only'; then
    echo "missing artifact repair hint" >&2
    exit 92
  fi
  cat <<'JSON'
{"meta":{"task_id":"task-pre-post-stall","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"recovered after pre-artifact then post-artifact stall","changeset":[]}
JSON
  exit 0
fi
echo "unexpected invocation $count" >&2
exit 93
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre/post stall stub: %v", err)
	}

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-pre-post-stall",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected combined pre/post stall recovery to succeed: %v", err)
	}
	if result.TaskResult.Summary != "recovered after pre-artifact then post-artifact stall" {
		t.Fatalf("unexpected result summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "3" {
		t.Fatalf("expected exactly 3 runner invocations, got %q", count)
	}
	last := diagnostics[len(diagnostics)-1]
	if last.Message != "runtime task retry completed" {
		t.Fatalf("expected retry completed diagnostic, got %q", last.Message)
	}
	if last.Fields["retry_status"] != "succeeded" {
		t.Fatalf("expected retry_status=succeeded, got %#v", last.Fields["retry_status"])
	}
	if last.Fields["last_stall_phase"] != "post_artifact" {
		t.Fatalf("expected last_stall_phase=post_artifact after real second stall, got %#v", last.Fields["last_stall_phase"])
	}
	if last.Fields["manifest_state_before_retry"] != "rich" {
		t.Fatalf("expected manifest_state_before_retry=rich, got %#v", last.Fields["manifest_state_before_retry"])
	}
}

func TestHeadlessRunnerPreArtifactRecoveryDoesNotWaitForLingeringStdoutPipe(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, 300*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-lingering-pipe.sh")
	script := `#!/bin/sh
set -eu
(trap '' TERM INT HUP; sleep 3) &
sleep 3
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write lingering-pipe stub: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-pre-stall-lingering",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	taskPayload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	args := buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))

	started := time.Now()
	_, _, _, runErr := runQwenCommand(context.Background(), task, commandPath, args, runQwenOptions{
		EnableCollectStallMonitor: true,
	})
	if runErr == nil {
		t.Fatalf("expected collect_stalled_before_artifacts sentinel")
	}
	if !errors.Is(runErr, errCollectStalledBeforeArtifacts) {
		t.Fatalf("expected collect_stalled_before_artifacts, got %v", runErr)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("expected forced retry to avoid long EOF wait, elapsed=%s", elapsed)
	}
}

func TestHeadlessRunnerPostArtifactRecoveryDoesNotWaitForLingeringStdoutPipe(t *testing.T) {
	setCollectStallTimingsForTest(t, 300*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-post-stall-lingering-pipe.sh")
	script := `#!/bin/sh
set -eu
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-overview.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
` + validRetryRichManifestJSON() + `
JSON
(trap '' TERM INT HUP; sleep 3) &
sleep 3
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write post-artifact lingering-pipe stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	task := acpruntime.Task{
		TaskID:       "task-post-stall-lingering",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	taskPayload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	args := buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))

	started := time.Now()
	_, _, _, runErr := runQwenCommand(context.Background(), task, commandPath, args, runQwenOptions{
		EnableCollectStallMonitor: true,
	})
	if runErr == nil {
		t.Fatalf("expected collect_stalled_after_artifacts sentinel")
	}
	if !errors.Is(runErr, errCollectStalledAfterArtifacts) {
		t.Fatalf("expected collect_stalled_after_artifacts, got %v", runErr)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("expected post-artifact retry path to avoid long EOF wait, elapsed=%s", elapsed)
	}
}

func TestHeadlessRunnerPreArtifactRecoveryKillsProcessTree(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, time.Second, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	childPIDFile := filepath.Join(tempDir, "child.pid")
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-child-tree.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
(trap '' TERM INT HUP; sleep 10) &
child=$!
printf '%%s' "$child" > %q
sleep 10
`, childPIDFile)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write process-tree stub: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-pre-stall-proctree",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	taskPayload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	args := buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))

	_, _, _, runErr := runQwenCommand(context.Background(), task, commandPath, args, runQwenOptions{
		EnableCollectStallMonitor: true,
	})
	if runErr == nil {
		t.Fatalf("expected collect_stalled_before_artifacts sentinel")
	}
	if !errors.Is(runErr, errCollectStalledBeforeArtifacts) {
		t.Fatalf("expected collect_stalled_before_artifacts, got %v", runErr)
	}

	var childPIDRaw []byte
	deadlineReadPID := time.Now().Add(2 * time.Second)
	for {
		childPIDRaw, err = os.ReadFile(childPIDFile)
		if err == nil {
			break
		}
		if time.Now().After(deadlineReadPID) {
			t.Fatalf("read child pid file: %v", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(childPIDRaw)))
	if err != nil {
		t.Fatalf("parse child pid: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		err = syscall.Kill(childPID, 0)
		if err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected stalled process tree child %d to be terminated", childPID)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestHeadlessRunnerDraftArtifactStallDoesNotWaitForLingeringStdoutPipe(t *testing.T) {
	setCollectStallTimingsForTest(t, 300*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-draft-stall-lingering-pipe.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
mkdir -p %q %q/charter %q/skills
cat > %q/constitution-draft.json <<'JSON'
%s
JSON
printf '# Constitution\n' > %q/charter/overview.md
printf 'version: 1\nprofiles: []\n' > %q/skills/subagents.yaml
(trap '' TERM INT HUP; sleep 3) &
sleep 3
`, writeRoot, draftRoot, draftRoot, writeRoot, validConstitutionDraftManifestJSON("run-1"), draftRoot, draftRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write draft lingering-pipe stub: %v", err)
	}

	task := acpruntime.Task{
		TaskID:            "task-draft-stall-lingering",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	}
	taskPayload, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	args := buildDefaultQwenArgs(task, buildPrompt(taskPayload, false))

	started := time.Now()
	_, _, _, runErr := runQwenCommand(context.Background(), task, commandPath, args, runQwenOptions{
		EnableDraftStallMonitor: true,
	})
	if runErr == nil {
		t.Fatalf("expected draft_stalled_after_artifacts sentinel")
	}
	if !errors.Is(runErr, errDraftStalledAfterArtifacts) {
		t.Fatalf("expected draft_stalled_after_artifacts, got %v", runErr)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("expected draft forced retry path to avoid long EOF wait, elapsed=%s", elapsed)
	}
}

func TestHeadlessRunnerRecoversStep0DraftStallAfterArtifactsViaRetry(t *testing.T) {
	setCollectStallTimingsForTest(t, 150*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	draftRoot := filepath.Join(tempDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	stateFile := filepath.Join(tempDir, "step0-draft-stall-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-step0-draft-stall-retry.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
mkdir -p %q %q/charter %q/skills
cat > %q/constitution-draft.json <<'JSON'
%s
JSON
printf '# Constitution\n' > %q/charter/overview.md
printf 'version: 1\nprofiles:\n  - id: baseline-architect\n    role: architect\n' > %q/skills/subagents.yaml
if [ "$count" -eq 1 ]; then
  sleep 10
  exit 0
fi
if ! echo "$last_arg" | grep -Fq 'Previous attempt stalled after writing draft artifacts'; then
  echo "missing draft stall retry hint" >&2
  exit 91
fi
if ! echo "$last_arg" | grep -Fq '"changeset": []'; then
  echo "missing empty changeset retry hint" >&2
  exit 92
fi
cat <<'JSON'
{"meta":{"task_id":"task-step0-draft-stall","step_id":"init.step0.constitution","run_id":"run-1","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-20T07:16:00Z"},"summary":"recovered after step0 draft stall","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, draftRoot, draftRoot, writeRoot, validConstitutionDraftManifestJSON("run-1"), draftRoot, draftRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write step0 draft stall retry stub: %v", err)
	}

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:            "task-step0-draft-stall",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         tempDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		StepContract:      "constitution",
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected step0 draft stall recovery to succeed: %v", err)
	}
	if result.TaskResult.Summary != "recovered after step0 draft stall" {
		t.Fatalf("unexpected repaired summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	if len(diagnostics) < 3 {
		t.Fatalf("expected draft stall diagnostics, got %d events", len(diagnostics))
	}
	if diagnostics[0].Message != "runtime task stalled after draft artifacts" {
		t.Fatalf("expected first diagnostic to report draft stall, got %q", diagnostics[0].Message)
	}
	if diagnostics[1].Message != "runtime task retry scheduled" {
		t.Fatalf("expected retry scheduled diagnostic, got %q", diagnostics[1].Message)
	}
	last := diagnostics[len(diagnostics)-1]
	if last.Message != "runtime task retry completed" {
		t.Fatalf("expected retry completed diagnostic, got %q", last.Message)
	}
	if last.Fields["retry_status"] != "succeeded" {
		t.Fatalf("expected retry_status=succeeded, got %#v", last.Fields["retry_status"])
	}
	if last.Fields["last_stall_phase"] != "post_artifact" {
		t.Fatalf("expected last_stall_phase=post_artifact, got %#v", last.Fields["last_stall_phase"])
	}
}

func TestHeadlessRunnerDoesNotTriggerPreArtifactStallWhileWriteRootMutates(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, time.Second, 20*time.Millisecond, 10*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-active-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  echo "retry-unexpected" >&2
  exit 97
fi
mkdir -p "$STALL_WRITE_ROOT"
for tick in 1 2 3 4 5; do
  printf 'tick-%s\n' "$tick" >> "$STALL_WRITE_ROOT/.progress.tmp"
  sleep 0.1
done
printf '# Extras\n' > "$STALL_WRITE_ROOT/extras-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "/tmp/write-root",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["active"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
cat <<'JSON'
{"meta":{"task_id":"task-pre-stall-active","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"completed with pre-artifact activity","changeset":[]}
JSON
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre-stall active stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	var diagnostics []acpruntime.DiagnosticEvent
	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-pre-stall-active",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		OnDiagnostic: func(event acpruntime.DiagnosticEvent) {
			diagnostics = append(diagnostics, event)
		},
	})
	if err != nil {
		t.Fatalf("expected write-root mutations to prevent pre-artifact stall: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-pre-stall-active" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
	for _, event := range diagnostics {
		if event.Message == "runtime task stalled before artifacts" {
			t.Fatalf("did not expect pre-artifact stall diagnostic while write_root mutations continued")
		}
	}
}

func TestHeadlessRunnerPreArtifactStallRetryFailurePersistsRawArtifactsAndClassifiesParseFailure(t *testing.T) {
	setCollectPreArtifactStallTimingsForTest(t, time.Second, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-pre-stall-invalid-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  printf 'not-json\n'
  exit 0
fi
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write pre-stall invalid retry stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-pre-stall-invalid",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse failure after pre-artifact stalled collect retry returned invalid output")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed, got %s", code)
	}
	if !strings.Contains(message, "collect_stalled_before_artifacts") {
		t.Fatalf("expected collect_stalled_before_artifacts in error, got %v", err)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output path in message, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(tempDir, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected raw output metadata under %s", filepath.Join(tempDir, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerCollectStallRetryFailurePersistsRawArtifactsAndClassifiesParseFailure(t *testing.T) {
	setCollectStallTimingsForTest(t, 150*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-stall-invalid-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  printf 'not-json\n'
  exit 0
fi
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "version": 1,
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.iac"]
    }
  ]
}
JSON
printf 'artifacts-ready\n'
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write invalid retry stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-stall-invalid",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse failure after stalled collect retry returned invalid output")
	}
	var runnerErr acpruntime.RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
	if runnerErr.Code != acpruntime.ErrorCodeRunnerParseFailed {
		t.Fatalf("expected runner_parse_failed, got %s", runnerErr.Code)
	}
	if !strings.Contains(runnerErr.Error(), "collect_stalled_after_artifacts") {
		t.Fatalf("expected collect_stalled_after_artifacts in error, got %v", runnerErr)
	}
	rawDir := filepath.Join(tempDir, "reports", "taskruns", "raw")
	entries, readErr := os.ReadDir(rawDir)
	if readErr != nil {
		t.Fatalf("read raw artifact dir: %v", readErr)
	}
	if len(entries) == 0 {
		t.Fatalf("expected raw artifacts under %s", rawDir)
	}
}

func TestHeadlessRunnerCollectStallArtifactRepairContractFailurePersistsRawArtifacts(t *testing.T) {
	setCollectStallTimingsForTest(t, 150*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-stall-skeletal-retry-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "version": 1,
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.iac"]
    }
  ]
}
JSON
  cat <<'JSON'
{"meta":{"task_id":"task-stall-skeletal","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"retry stayed skeletal","changeset":[]}
JSON
  exit 0
fi
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "version": 1,
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.iac"]
    }
  ]
}
JSON
printf 'artifacts-ready\n'
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write skeletal retry stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-stall-skeletal",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected parse failure after stalled collect retry kept skeletal artifacts")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if !strings.Contains(message, "collect_stalled_after_artifacts") {
		t.Fatalf("expected collect_stalled_after_artifacts in message, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output path in message, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(tempDir, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected raw output metadata under %s", filepath.Join(tempDir, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerCollectArtifactRepairFailsOnInvalidCompatibilityManifest(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	stateFile := filepath.Join(tempDir, "qwen-invalid-compatibility-count.txt")
	commandPath := filepath.Join(tempDir, "qwen-invalid-compatibility.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
mkdir -p %q
printf '# Extras\n' > %q/extras-analysis.md
if [ "$count" -eq 1 ]; then
  cat > %q/shard-pack-manifest.json <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["invalid entities"]},
    "questions": [],
    "entities": [{"id":"svc.extras","type":"service","name":"Extras"}],
    "edges": [],
    "findings": ["finding as string"]
  }
}
MANIFEST
  cat <<'JSON'
{"meta":{"task_id":"task-invalid-compatibility","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"collect returned invalid compatibility manifest","changeset":[]}
JSON
  exit 0
fi
if ! echo "$last_arg" | grep -Fq 'compatibility.entities[*] MUST remain full entity objects with provenance'; then
  echo "missing compatibility entity repair hint" >&2
  exit 97
fi
if ! echo "$last_arg" | grep -Fq 'compatibility.findings[*] MUST remain objects'; then
  echo "missing findings object repair hint" >&2
  exit 98
fi
if ! echo "$last_arg" | grep -Fq 'compatibility.edges[*] MUST remain objects with canonical keys type/from/to'; then
  echo "missing edge canonical-key repair hint" >&2
  exit 99
fi
if ! echo "$last_arg" | grep -Fq 'each finding MUST include title'; then
  echo "missing findings title repair hint" >&2
  exit 100
fi
cat > %q/shard-pack-manifest.json <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "refresh.step1.collect",
  "shard_id": "bank-of-anthos-extras",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["extras"],
  "documents": [
    {
      "id": "doc.extras",
      "kind": "analysis",
      "title": "Extras",
      "path": "extras-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/extras-analysis.md",
      "topics": ["extras"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.extras.readme"],
      "document_ids": ["doc.extras"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["extras"], "missing": ["owner mappings"], "notes": ["still invalid"]},
    "questions": [],
    "entities": [{"id":"svc.extras","type":"service","name":"Extras"}],
    "edges": [{"id":"edge.extras.dep","kind":"depends_on","source":"svc.extras","target":"svc.payments"}],
    "findings": [{"id":"finding.owner.missing","severity":"medium","description":"still missing title","rule_id":"rule.owner.required","related_ids":["svc.extras"],"provenance":{"kind":"inference","confidence":0.6,"evidence":[{"repo":"bank-of-anthos","path":"README.md"}]}}]
  }
}
MANIFEST
cat <<'JSON'
{"meta":{"task_id":"task-invalid-compatibility","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"repair kept invalid compatibility manifest","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, writeRoot, writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write invalid compatibility stub: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-invalid-compatibility",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/bank-of-anthos-extras",
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected repair failure for invalid compatibility manifest")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if !strings.Contains(message, "collect artifacts remained invalid after one repair attempt") {
		t.Fatalf("expected contract failure context, got %q", message)
	}
	if !strings.Contains(message, "shard pack manifest is invalid") {
		t.Fatalf("expected explicit manifest validation reason, got %q", message)
	}
	if !strings.Contains(strings.ToLower(message), "title") &&
		!strings.Contains(strings.ToLower(message), "provenance") &&
		!strings.Contains(strings.ToLower(message), "type") {
		t.Fatalf("expected explicit nested manifest contract reason, got %q", message)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output diagnostics, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(tempDir, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected raw output metadata under %s", filepath.Join(tempDir, "reports", "taskruns", "raw"))
	}
}

func TestHeadlessRunnerPreservesContextCancellation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-cancel-stub.sh")
	script := `#!/bin/sh
set -eu
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write cancel command: %v", err)
	}

	runner := HeadlessRunner{
		Command: commandPath,
		Args:    []string{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)

	_, err := runner.Run(ctx, acpruntime.Task{
		TaskID:       "task-cancel",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected error to preserve context.Canceled, got %v", err)
	}
}

func TestHeadlessRunnerParsesTaskResultFromResponseField(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"response":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen response parser: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerParsesTaskResultFromJsonArrayResultMessage(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '[{"type":"system","subtype":"session_start"},{"type":"result","result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step3.findings\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}]'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen array parser: %v", err)
	}
	if result.TaskResult.Meta.StepID != "init.step3.findings" {
		t.Fatalf("unexpected step id %q", result.TaskResult.Meta.StepID)
	}
}

func TestHeadlessRunnerParsesTaskResultFromQwenAssistantContentEvents(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '[{"type":"system","subtype":"init"},{"type":"assistant","message":{"content":[{"type":"thinking","thinking":"ignored"},{"type":"text","text":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}" }]}}]'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen event content parser: %v", err)
	}
	if result.TaskResult.Meta.Runtime.Name != "qwen-code" {
		t.Fatalf("unexpected runtime name %q", result.TaskResult.Meta.Runtime.Name)
	}
}

func TestHeadlessRunnerParsesTaskResultFromJsonObjectsStream(t *testing.T) {
	t.Parallel()

	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n%s\n' '{"type":"system","message":"ignored"}' '{"type":"result","result":"{\"meta\":{\"task_id\":\"task-1\",\"step_id\":\"init.step1.collect\",\"runtime\":{\"name\":\"qwen-code\",\"version\":\"test\"},\"started_at\":\"2026-04-03T12:00:00Z\"},\"summary\":\"ok\",\"changeset\":[]}"}'`,
		},
	}

	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-1",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("run qwen stream parser: %v", err)
	}
	if result.TaskResult.Meta.TaskID != "task-1" {
		t.Fatalf("unexpected task id %q", result.TaskResult.Meta.TaskID)
	}
}

func TestHeadlessRunnerBindingMismatchClassifiesAsParseFailed(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	runner := HeadlessRunner{
		Command: "sh",
		Args: []string{
			"-c",
			`printf '%s\n' '{"meta":{"task_id":"task-stale","step_id":"init.step1.collect","run_id":"run-stale","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}'`,
		},
	}

	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-expected",
		RunID:        "run-expected",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected binding parse-failed error")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok {
		t.Fatalf("expected classify error to succeed")
	}
	if code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("unexpected error code %q", code)
	}
	if !strings.Contains(message, "parse_stage=binding") {
		t.Fatalf("expected parse_stage=binding in parse-fail message, got %q", message)
	}
}

func TestBuildPromptIncludesStepSpecificPolicies(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh",
		RunID:        "run-1",
		StepID:       "refresh.step3.findings",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "STEP POLICY refresh.step3.findings") {
		t.Fatalf("expected step3 policy block in prompt")
	}
	if !strings.Contains(prompt, "include at least one add_finding operation") {
		t.Fatalf("expected required add_finding policy in prompt")
	}
	if !strings.Contains(prompt, "coverage.missing MUST use canonical terms only") {
		t.Fatalf("expected canonical coverage dictionary policy in prompt")
	}
	if !strings.Contains(prompt, "For upsert_edge use canonical keys only: edge.id, edge.type, edge.from, edge.to.") {
		t.Fatalf("expected canonical upsert_edge key policy in prompt")
	}
	if !strings.Contains(prompt, "Forbidden edge aliases: edge.kind, edge.source, edge.target.") {
		t.Fatalf("expected forbidden edge alias policy in prompt")
	}
}

func TestBuildPromptRefreshStep1CollectIncludesNoWebSearchPolicy(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-refresh-step1",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "do NOT perform web search or external browsing") {
		t.Fatalf("expected no-web-search rule in step1 collect policy")
	}
	if !strings.Contains(prompt, "Do NOT emit synthetic evidence paths such as search_source/*") {
		t.Fatalf("expected synthetic evidence-path ban in step1 collect policy")
	}
}

func TestBuildPromptCollectIncludesExistingEntrypointHintsAndDelegationBan(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	repoRoot := filepath.Join(workspaceDir, "service-repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	packageJSONPath := filepath.Join(repoRoot, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte("{\"name\":\"service-repo\"}\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	task := acpruntime.Task{
		TaskID:           "task-init-step1",
		RunID:            "run-1",
		StepID:           "init.step1.collect",
		Workspace:        workspaceDir,
		ReadContextRoots: []string{repoRoot},
		RepoScopes:       []string{"service-repo"},
		StartedAtUTC:     time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "Do NOT delegate to agent/subagent helpers.") {
		t.Fatalf("expected delegation ban in collect prompt")
	}
	if strings.Contains(prompt, `"op": "upsert_entity"`) {
		t.Fatalf("did not expect synthetic upsert_entity template in collect prompt")
	}
	if !strings.Contains(prompt, filepath.ToSlash(packageJSONPath)) {
		t.Fatalf("expected existing package.json entrypoint hint in prompt")
	}
	if strings.Contains(prompt, filepath.ToSlash(filepath.Join(repoRoot, "README.md"))) {
		t.Fatalf("did not expect non-existent README.md entrypoint hint in prompt")
	}
}

func TestBuildPromptConstitutionIncludesExistingEntrypointHintsAndDelegationBan(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	repoRoot := filepath.Join(workspaceDir, "service-repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("mkdir repo root: %v", err)
	}
	packageJSONPath := filepath.Join(repoRoot, "package.json")
	if err := os.WriteFile(packageJSONPath, []byte("{\"name\":\"service-repo\"}\n"), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	task := acpruntime.Task{
		TaskID:           "task-init-step0",
		RunID:            "run-1",
		StepID:           "init.step0.constitution",
		Workspace:        workspaceDir,
		ReadContextRoots: []string{repoRoot},
		RepoScopes:       []string{"service-repo"},
		StartedAtUTC:     time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "STEP POLICY init.step0.constitution:") {
		t.Fatalf("expected step0 policy section in prompt")
	}
	if !strings.Contains(prompt, "Do NOT delegate to agent/subagent helpers.") {
		t.Fatalf("expected delegation ban in constitution prompt")
	}
	if !strings.Contains(prompt, "Write constitution-draft.json in write_root.") {
		t.Fatalf("expected constitution draft manifest instruction in prompt")
	}
	if !strings.Contains(prompt, `"step_contract": "constitution"`) {
		t.Fatalf("expected exact constitution draft manifest example in prompt")
	}
	if !strings.Contains(prompt, `"canonical_path": "charter/overview.md"`) || !strings.Contains(prompt, `"canonical_path": "skills/subagents.yaml"`) {
		t.Fatalf("expected exact constitution outputs mapping in prompt")
	}
	if !strings.Contains(prompt, "draft_final_root/charter-overview.md") || !strings.Contains(prompt, "draft_final_root/baseline-subagents.yaml") {
		t.Fatalf("expected exact draft file placement guidance in prompt")
	}
	if !strings.Contains(prompt, "Do NOT place the draft files under draft_final_root/charter/ or draft_final_root/skills/") {
		t.Fatalf("expected anti-nested draft file guidance in prompt")
	}
	if strings.Contains(prompt, `"op": "upsert_entity"`) {
		t.Fatalf("did not expect synthetic upsert_entity template for draft-only step0")
	}
	if !strings.Contains(prompt, filepath.ToSlash(packageJSONPath)) {
		t.Fatalf("expected existing package.json entrypoint hint in prompt")
	}
	if strings.Contains(prompt, filepath.ToSlash(filepath.Join(repoRoot, "README.md"))) {
		t.Fatalf("did not expect non-existent README.md entrypoint hint in prompt")
	}
}

func TestBuildPromptConstitutionRetryUsesDraftSpecificRecoveryHints(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	writeRoot := filepath.Join(workspaceDir, "write-root")
	draftRoot := filepath.Join(workspaceDir, "draft-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.MkdirAll(draftRoot, 0o755); err != nil {
		t.Fatalf("mkdir draft root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "constitution-draft.json"), []byte(loadRuntimeFixture(t, "qwen_step0_bank_single_legacy_constitution_draft.json")), 0o644); err != nil {
		t.Fatalf("write legacy constitution draft: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "charter-overview.md"), []byte("# Constitution\n"), 0o644); err != nil {
		t.Fatalf("write charter overview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(draftRoot, "baseline-subagents.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write baseline subagents: %v", err)
	}

	task := acpruntime.Task{
		TaskID:            "task-step0-retry",
		RunID:             "run-1",
		StepID:            "init.step0.constitution",
		Workspace:         workspaceDir,
		WriteRoot:         writeRoot,
		DraftFinalRoot:    draftRoot,
		ExpectedArtifacts: []string{"constitution-draft.json"},
		RepoScopes:        []string{"bank-of-anthos"},
		StartedAtUTC:      time.Date(2026, 4, 20, 7, 16, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, true)
	if !strings.Contains(prompt, "constitution-draft.json") {
		t.Fatalf("expected constitution draft manifest recovery guidance in prompt")
	}
	if !strings.Contains(prompt, "charter/overview.md") || !strings.Contains(prompt, "skills/subagents.yaml") {
		t.Fatalf("expected constitution canonical output mapping in retry prompt")
	}
	if !strings.Contains(prompt, "draft_final_root/charter-overview.md") || !strings.Contains(prompt, "draft_final_root/baseline-subagents.yaml") {
		t.Fatalf("expected exact draft file placement guidance in retry prompt")
	}
	if strings.Contains(prompt, "shard-pack-manifest.json") {
		t.Fatalf("did not expect shard-pack-manifest wording in step0 retry prompt")
	}
	if strings.Contains(prompt, "shard artifacts") {
		t.Fatalf("did not expect shard-artifacts wording in step0 retry prompt")
	}
}

func TestBuildParseRepairHintsForStep0UseDraftManifestRule(t *testing.T) {
	t.Parallel()

	hints := strings.Join(buildParseRepairHints(
		"init.step0.constitution",
		"schema",
		errors.New(`additionalProperties 'compatibility' not allowed`),
	), "\n")
	if !strings.Contains(hints, "constitution-draft.json / asis-draft-manifest.json / proposals-draft-manifest.json") {
		t.Fatalf("expected draft-manifest compatibility rule in hints, got %q", hints)
	}
	if strings.Contains(hints, "shard-pack-manifest.json") {
		t.Fatalf("did not expect collect manifest wording in step0 parse-repair hints")
	}
}

func TestBuildPromptIncludesStrictResultJsonAndFinalResponseDiscipline(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-result-discipline",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, true)
	if !strings.Contains(prompt, "STRICT RESULT JSON MODE") {
		t.Fatalf("expected strict result json mode block in prompt")
	}
	if !strings.Contains(prompt, "Prefer returning a direct TaskResult JSON object") {
		t.Fatalf("expected direct TaskResult preference in prompt")
	}
	if !strings.Contains(prompt, "Do NOT narrate file writes, manifest contents, or planning steps in the final message.") {
		t.Fatalf("expected final response discipline in prompt")
	}
	if !strings.Contains(prompt, `Final response MUST start with "{" and end with "}".`) {
		t.Fatalf("expected retry json framing guard in prompt")
	}
}

func TestBuildPromptRetryIncludesWriteRootRecoveryGuidance(t *testing.T) {
	t.Parallel()

	writeRoot := filepath.Join(t.TempDir(), "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	for name, contents := range map[string]string{
		"shard-pack-manifest.json": validRetryRichManifestJSON(),
		"iac-overview.md":          "# Overview\n",
		"iac-analysis.md":          "# Analysis\n",
	} {
		if err := os.WriteFile(filepath.Join(writeRoot, name), []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	task := acpruntime.Task{
		TaskID:       "task-retry-recovery",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, true)
	if !strings.Contains(prompt, "RETRY RECOVERY MODE") {
		t.Fatalf("expected retry recovery block in prompt")
	}
	if !strings.Contains(prompt, "Retry goal is JSON repair, not fresh repository exploration.") {
		t.Fatalf("expected json-repair guidance in prompt")
	}
	if !strings.Contains(prompt, "Do NOT use todo_write") {
		t.Fatalf("expected todo_write ban in retry prompt")
	}
	if !strings.Contains(prompt, "shard-pack-manifest.json is already present in write_root") {
		t.Fatalf("expected manifest reuse guidance in retry prompt")
	}
	if !strings.Contains(prompt, `Do NOT collapse a multi-document refresh into one generic "cite.runtime-summary" citation.`) {
		t.Fatalf("expected runtime-summary collapse ban in retry prompt")
	}
	if !strings.Contains(prompt, "Preserve repo-specific citations when repository evidence already exists or can be recovered from repo roots.") {
		t.Fatalf("expected repo-specific citation preservation rule in retry prompt")
	}
	if !strings.Contains(prompt, `Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`) {
		t.Fatalf("expected top-level compatibility ban in retry prompt")
	}
	if !strings.Contains(prompt, `Prefer "changeset": [] when write_root already contains authored docs`) {
		t.Fatalf("expected minimal changeset guidance in retry prompt")
	}
	if !strings.Contains(prompt, "Do NOT copy long citations, topic arrays, or manifest document inventories into TaskResult JSON.") {
		t.Fatalf("expected manifest-copy ban in retry prompt")
	}
	if !strings.Contains(prompt, "Retry-safe minimal template (preferred when reusing existing write_root artifacts):") {
		t.Fatalf("expected retry-safe minimal template block in prompt")
	}
	if !strings.Contains(prompt, `"changeset": []`) {
		t.Fatalf("expected empty changeset in retry-safe template, got %q", prompt)
	}
	if !strings.Contains(prompt, "iac-analysis.md") || !strings.Contains(prompt, "iac-overview.md") {
		t.Fatalf("expected write_root snapshot filenames in retry prompt, got %q", prompt)
	}
}

func TestBuildDefaultQwenArgsUsesPromptFlagWithWorkspaceAndRepoDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "payments-service")
	for _, dir := range []string{workspace, repoPath} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: payments-service\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	args := buildDefaultQwenArgs(task, "prompt-text")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--include-directories "+workspace) {
		t.Fatalf("expected workspace include-directories in args, got %q", joined)
	}
	if !strings.Contains(joined, "--include-directories "+repoPath) {
		t.Fatalf("expected repo include-directories in args, got %q", joined)
	}
	if !strings.Contains(joined, "--chat-recording false") {
		t.Fatalf("expected chat recording to be disabled in args, got %q", joined)
	}
	if !strings.Contains(joined, "--prompt prompt-text") {
		t.Fatalf("expected --prompt usage in args, got %q", joined)
	}
	if args[len(args)-2] != "--prompt" || args[len(args)-1] != "prompt-text" {
		t.Fatalf("expected prompt to be appended as --prompt <value>, got %#v", args)
	}
}

func TestBuildRetryQwenArgsConstrainsCollectRetryToWriteRootWhenArtifactsExist(t *testing.T) {
	t.Parallel()

	writeRoot := filepath.Join(t.TempDir(), "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "iac-overview.md"), []byte("# Overview\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-retry-args",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}

	args := buildRetryQwenArgs(task, "prompt-text")
	includeDirs := []string{}
	for idx := 0; idx < len(args)-1; idx++ {
		if args[idx] == "--include-directories" {
			includeDirs = append(includeDirs, args[idx+1])
		}
	}
	if len(includeDirs) != 1 || includeDirs[0] != writeRoot {
		t.Fatalf("expected retry include-directories to be constrained to write_root, got %#v", includeDirs)
	}
}

func TestBuildRetryQwenArgsKeepsRepoRootsWhenManifestIsSkeletal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetrySkeletalManifestJSON()), 0o644); err != nil {
		t.Fatalf("write skeletal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-analysis.md"), []byte("# Analysis\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-retry-skeletal",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}

	args := buildRetryQwenArgs(task, "prompt-text")
	includeDirs := []string{}
	for idx := 0; idx < len(args)-1; idx++ {
		if args[idx] == "--include-directories" {
			includeDirs = append(includeDirs, args[idx+1])
		}
	}
	if len(includeDirs) < 2 {
		t.Fatalf("expected retry include-directories to keep workspace and repo roots, got %#v", includeDirs)
	}
	if includeDirs[0] != workspace {
		t.Fatalf("expected workspace include-directories first, got %#v", includeDirs)
	}
	if includeDirs[1] != repoPath {
		t.Fatalf("expected repo include-directories to remain available, got %#v", includeDirs)
	}
}

func TestBuildRetryQwenArgsKeepsRepoRootsWhenRepoSpecificCitationIsUnlinked(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryUnlinkedRepoSpecificManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest with unlinked repo-specific citation: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-analysis.md"), []byte("# Analysis\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-retry-unlinked-citation",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}

	args := buildRetryQwenArgs(task, "prompt-text")
	includeDirs := []string{}
	for idx := 0; idx < len(args)-1; idx++ {
		if args[idx] == "--include-directories" {
			includeDirs = append(includeDirs, args[idx+1])
		}
	}
	if len(includeDirs) < 2 {
		t.Fatalf("expected retry include-directories to keep workspace and repo roots for unlinked citations, got %#v", includeDirs)
	}
	if includeDirs[0] != workspace {
		t.Fatalf("expected workspace include-directories first, got %#v", includeDirs)
	}
	if includeDirs[1] != repoPath {
		t.Fatalf("expected repo include-directories to remain available, got %#v", includeDirs)
	}
}

func TestBuildPromptRetryWarnsWhenManifestIsSkeletal(t *testing.T) {
	t.Parallel()

	writeRoot := filepath.Join(t.TempDir(), "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetrySkeletalManifestJSON()), 0o644); err != nil {
		t.Fatalf("write skeletal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-analysis.md"), []byte("# Analysis\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-retry-skeletal-prompt",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, true)
	if !strings.Contains(prompt, "looks skeletal/reuse-only; keep repo roots in include-directories") {
		t.Fatalf("expected skeletal manifest guidance in retry prompt")
	}
	if !strings.Contains(prompt, `Do NOT reduce multi-document refresh evidence to one generic "cite.runtime-summary" citation.`) {
		t.Fatalf("expected collapse warning for skeletal manifest prompt")
	}
}

func TestBuildPromptIncludesCanonicalManifestSchemaGuardrails(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-manifest-schema-guardrails",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    t.TempDir(),
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs",
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, `Do NOT use documents[].citations; only documents[].citation_ids is allowed.`) {
		t.Fatalf("expected canonical manifest field guardrail in prompt")
	}
	if !strings.Contains(prompt, `citations[].claim_ids MUST be globally unique across the assembled staged final set`) {
		t.Fatalf("expected global claim-id uniqueness guardrail in prompt")
	}
	if !strings.Contains(prompt, `In shard-pack-manifest.json, compatibility MUST include coverage, questions, entities, edges, and findings`) {
		t.Fatalf("expected manifest-scoped compatibility completeness guardrail in prompt")
	}
	if !strings.Contains(prompt, `Do NOT use reports/taskruns/... staging paths as canonical_path.`) {
		t.Fatalf("expected canonical_path staging-path ban in prompt")
	}
	if !strings.Contains(prompt, `documents[].path MUST be artifact_root-relative only`) {
		t.Fatalf("expected artifact_root-relative documents[].path guardrail in prompt")
	}
	if !strings.Contains(prompt, `compatibility.entities[*] MUST remain full entity objects with provenance`) {
		t.Fatalf("expected compatibility entity provenance guardrail in prompt")
	}
	if !strings.Contains(prompt, `compatibility.findings[*] MUST remain structured finding objects`) {
		t.Fatalf("expected compatibility findings object guardrail in prompt")
	}
}

func TestBuildPromptInjectsWorkspacePromptPackAfterPolicyLayer(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	packDir := filepath.Join(workspaceDir, "skills", "prompt-packs")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatalf("mkdir prompt pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "collect-context.md"), []byte("Workspace collect pack guidance.\n"), 0o644); err != nil {
		t.Fatalf("write collect prompt pack: %v", err)
	}

	task := acpruntime.Task{
		TaskID:       "task-workspace-pack",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspaceDir,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	prompt := buildPrompt(raw, false)
	if !strings.Contains(prompt, "Workspace collect pack guidance.") {
		t.Fatalf("expected workspace prompt-pack content in prompt")
	}
	stepIdx := strings.Index(prompt, "STEP POLICY init.step1.collect:")
	packIdx := strings.Index(prompt, "WORKSPACE PROMPT PACK CONTENT LAYER:")
	templateIdx := strings.Index(prompt, "Schema-valid template for this task")
	if stepIdx == -1 || packIdx == -1 || templateIdx == -1 {
		t.Fatalf("expected prompt sections to exist, got:\n%s", prompt)
	}
	if !(stepIdx < packIdx && packIdx < templateIdx) {
		t.Fatalf("expected workspace prompt pack to appear after policy layer and before provider footer/template")
	}
}

func TestBuildPromptRetryIncludesSchemaFailureHintsForInvalidChangesetOp(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-invalid-op-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    filepath.Join(t.TempDir(), "write-root"),
		RepoScopes:   []string{"course-discovery"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	parseErr := errors.New(`taskresult is invalid: taskresult.schema.json validation failed: jsonschema: '/changeset/0/op' does not validate`)
	prompt := buildPromptWithModeAndHints(raw, promptRetryParse, buildParseRepairHints(task.StepID, "schema", parseErr))
	if !strings.Contains(prompt, "Previous schema validation failure") {
		t.Fatalf("expected schema failure hint in retry prompt")
	}
	if !strings.Contains(prompt, "Unknown changeset[].op values are forbidden") {
		t.Fatalf("expected allowed-op whitelist in retry prompt")
	}
	if !strings.Contains(prompt, `For op="add_doc_artifact", the payload key MUST be "doc_artifact"; never use "artifact".`) {
		t.Fatalf("expected doc_artifact payload guardrail in retry prompt")
	}
	if !strings.Contains(prompt, "Do NOT use ad-hoc ops such as upsert_file") {
		t.Fatalf("expected upsert_file ban in retry prompt")
	}
}

func TestBuildPromptRetryIncludesTopLevelCompatibilityBanForSchemaFailure(t *testing.T) {
	t.Parallel()

	task := acpruntime.Task{
		TaskID:       "task-invalid-compatibility-retry",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    filepath.Join(t.TempDir(), "write-root"),
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	parseErr := errors.New(`taskresult is invalid: taskresult.schema.json validation failed: jsonschema: '' does not validate with https://example.local/acp/schemas/taskresult.schema.json#/additionalProperties: additionalProperties 'compatibility' not allowed`)
	prompt := buildPromptWithModeAndHints(raw, promptRetryParse, buildParseRepairHints(task.StepID, "schema", parseErr))
	if !strings.Contains(prompt, `Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.`) {
		t.Fatalf("expected top-level compatibility ban in schema retry prompt")
	}
}

func TestHeadlessRunnerRecoversCollectStallRetryWithTopLevelCompatibilityBan(t *testing.T) {
	setCollectStallTimingsForTest(t, 150*time.Millisecond, 25*time.Millisecond, 25*time.Millisecond)

	tempDir := t.TempDir()
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	commandPath := filepath.Join(tempDir, "qwen-stall-compatibility-ban-stub.sh")
	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -q "RETRY RECOVERY MODE"; then
  if echo "$last_arg" | grep -Fq 'Do NOT add top-level "compatibility" to TaskResult JSON; keep compatibility only inside shard-pack-manifest.json.' \
    && echo "$last_arg" | grep -Fq 'In shard-pack-manifest.json, compatibility.coverage/questions/entities/edges/findings must all exist'; then
    cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'MANIFEST'
{
  "version": 1,
  "run_id": "run-1",
  "step_id": "init.step1.collect",
  "shard_id": "bank-of-anthos-iac",
  "agent_role": "shard-analyst",
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "repo_scopes": ["bank-of-anthos"],
  "path_scopes": ["iac"],
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.bank.readme"]
    }
  ],
  "citations": [
    {
      "id": "cite.bank.readme",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.iac.readme"],
      "document_ids": ["doc.iac"]
    }
  ],
  "compatibility": {
    "coverage": {"observed": ["iac"], "missing": ["owner mappings"], "notes": ["recovered"]},
    "questions": [],
    "entities": [],
    "edges": [],
    "findings": []
  }
}
MANIFEST
    cat <<'JSON'
{"meta":{"task_id":"task-stall-compatibility-ban","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"recovered without top-level compatibility","changeset":[]}
JSON
    exit 0
  fi
  cat <<'JSON'
{"meta":{"task_id":"task-stall-compatibility-ban","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"invalid compatibility response","changeset":[],"compatibility":{"coverage":{"observed":["artifacts"],"missing":["owner mappings"],"notes":["invalid"]},"questions":[],"entities":[],"edges":[],"findings":[]}}
JSON
  exit 0
fi
mkdir -p "$STALL_WRITE_ROOT"
printf '# IAC\n' > "$STALL_WRITE_ROOT/iac-analysis.md"
cat > "$STALL_WRITE_ROOT/shard-pack-manifest.json" <<'JSON'
{
  "artifact_root": "reports/taskruns/run-1/staging/shards/bank-of-anthos-iac",
  "version": 1,
  "documents": [
    {
      "id": "doc.iac",
      "kind": "analysis",
      "title": "IAC",
      "path": "iac-analysis.md",
      "canonical_path": "reports/as-is/bank-of-anthos/iac-analysis.md",
      "topics": ["iac"],
      "citation_ids": ["cite.runtime-summary"]
    }
  ],
  "citations": [
    {
      "id": "cite.runtime-summary",
      "repo": "bank-of-anthos",
      "path": "README.md",
      "claim_ids": ["claim.runtime.summary"],
      "document_ids": ["doc.iac"]
    }
  ]
}
JSON
printf 'artifacts-ready\n'
sleep 10
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write compatibility-ban retry stub: %v", err)
	}
	t.Setenv("STALL_WRITE_ROOT", writeRoot)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-stall-compatibility-ban",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    tempDir,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected compatibility-ban retry to succeed: %v", err)
	}
	if result.TaskResult.Summary != "recovered without top-level compatibility" {
		t.Fatalf("unexpected repaired result summary %q", result.TaskResult.Summary)
	}
}

func TestHeadlessRunnerRetriesCapturedLiveOpenedxInvalidChangesetOpFixture(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-openedx-invalid-op-retry-stub.sh")
	fixturePath := filepath.Join(tempDir, "openedx-invalid-op-output.txt")
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_openedx_invalid_changeset_op_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write openedx invalid-op fixture copy: %v", err)
	}
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "iac-overview.md"), []byte("# Overview\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	script := `#!/bin/sh
set -eu
last_arg=""
for arg in "$@"; do
  last_arg="$arg"
done
if echo "$last_arg" | grep -Fq "Unknown changeset[].op values are forbidden" \
  && echo "$last_arg" | grep -Fq "Do NOT use ad-hoc ops such as upsert_file"; then
  cat <<'JSON'
{"meta":{"task_id":"task-openedx-invalid-op","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"schema retry fixed invalid op","changeset":[]}
JSON
  exit 0
fi
cat "$QWEN_OPENEDX_INVALID_OP_FIXTURE"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write openedx invalid-op retry command: %v", err)
	}

	t.Setenv("QWEN_OPENEDX_INVALID_OP_FIXTURE", fixturePath)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-openedx-invalid-op",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"course-discovery"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected retry run to succeed after invalid-op fixture: %v", err)
	}
	if result.TaskResult.Summary != "schema retry fixed invalid op" {
		t.Fatalf("unexpected result summary %q", result.TaskResult.Summary)
	}
}

func TestHeadlessRunnerNormalizesCapturedLiveBankExtrasLegacyDocArtifactFixture(t *testing.T) {
	tempDir := t.TempDir()
	commandPath := filepath.Join(tempDir, "qwen-bank-extras-legacy-doc-artifact-stub.sh")
	fixturePath := filepath.Join(tempDir, "bank-extras-legacy-doc-artifact-output.txt")
	writeRoot := filepath.Join(tempDir, "write-root")
	if err := os.WriteFile(fixturePath, []byte(loadCapturedLiveStdoutFixture(t, "qwen_live_bank_extras_legacy_doc_artifact_stdout.txt")), 0o644); err != nil {
		t.Fatalf("write bank extras fixture copy: %v", err)
	}
	if err := os.MkdirAll(writeRoot, 0o755); err != nil {
		t.Fatalf("mkdir write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "shard-pack-manifest.json"), []byte(validRetryRichManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(writeRoot, "extras-overview.md"), []byte("# Extras\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}

	script := `#!/bin/sh
set -eu
cat "$QWEN_BANK_EXTRAS_LEGACY_DOC_ARTIFACT_FIXTURE"
`
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write bank extras legacy artifact command: %v", err)
	}

	t.Setenv("QWEN_BANK_EXTRAS_LEGACY_DOC_ARTIFACT_FIXTURE", fixturePath)

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-run_20260418_214016_001-init-step1-collect-domain-bank-of-anthos-shard-bank-of-anthos-extras",
		RunID:        "run_20260418_214016_001",
		StepID:       "init.step1.collect",
		Workspace:    t.TempDir(),
		WriteRoot:    writeRoot,
		RepoScope:    "bank-of-anthos",
		RepoScopes:   []string{"bank-of-anthos"},
		PathScopes:   []string{"extras"},
		StartedAtUTC: time.Date(2026, 4, 18, 21, 48, 51, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected bank extras legacy artifact fixture to normalize successfully: %v", err)
	}
	if len(result.TaskResult.Changeset) != 0 {
		t.Fatalf("expected malformed manifest repair op to be dropped, got %#v", result.TaskResult.Changeset)
	}
}

func TestHeadlessRunnerRejectsInvalidManifestAfterArtifactRepairFixture(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "cinder")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: cinder\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	initialManifest := loadCapturedLiveStdoutFixture(t, "openstack_cinder_invalid_manifest_initial.json")
	repairedManifest := loadCapturedLiveStdoutFixture(t, "openstack_cinder_invalid_manifest_repaired.json")
	commandPath := filepath.Join(root, "qwen-invalid-manifest-repair.sh")
	stateFile := filepath.Join(root, "qwen-invalid-manifest-repair-count.txt")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
cat <<'EOF' > %q/cinder-as-is.md
# Cinder
EOF
if [ "$count" -eq 1 ]; then
  cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
  cat <<'JSON'
{"meta":{"task_id":"task-openstack-invalid-manifest","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"initial collect","changeset":[]}
JSON
  exit 0
fi
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'JSON'
{"meta":{"task_id":"task-openstack-invalid-manifest","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"repair stayed invalid","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, writeRoot, initialManifest, writeRoot, repairedManifest)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write invalid manifest repair command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-openstack-invalid-manifest",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"cinder"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected invalid repaired manifest fixture to fail")
	}
	code, _, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	if restored := string(mustReadFile(t, filepath.Join(writeRoot, "shard-pack-manifest.json"))); strings.Contains(restored, `"entities": true`) {
		t.Fatalf("expected write_root snapshot restore to discard invalid repaired manifest, got %q", restored)
	}
}

func TestHeadlessRunnerRepairsSkeletalCollectArtifactsAfterSchemaValidRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "qwen-repair-count.txt")
	commandPath := filepath.Join(root, "qwen-artifact-repair.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
if [ "$count" -eq 1 ]; then
  cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
  cat <<'EOF' > %q/shard-analysis.md
# Reused
EOF
  cat <<'JSON'
{"meta":{"task_id":"task-artifact-repair","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"initial collect","changeset":[]}
JSON
  exit 0
fi
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'EOF' > %q/iac-overview.md
# IAC Overview
EOF
cat <<'JSON'
{"meta":{"task_id":"task-artifact-repair","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"artifact repair succeeded","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, validRetrySkeletalManifestJSON(), writeRoot, writeRoot, validRetryRichManifestJSON(), writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write repair command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-artifact-repair",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected artifact repair retry to succeed: %v", err)
	}
	if result.TaskResult.Summary != "artifact repair succeeded" {
		t.Fatalf("expected repaired result summary, got %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	assessment, assessErr := assessRetryManifestAtWriteRoot(writeRoot)
	if assessErr != nil {
		t.Fatalf("assess repaired manifest: %v", assessErr)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired manifest to be rich, got %#v", assessment)
	}
}

func TestHeadlessRunnerRestoresOriginalArtifactsWhenRepairDoesNotImproveManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "qwen-repair-restore-count.txt")
	commandPath := filepath.Join(root, "qwen-artifact-restore.sh")
	initialDoc := "# Original skeletal analysis\n"
	retryDoc := "# Retry overwrite that should be rolled back\n"
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
if [ "$count" -eq 1 ]; then
  cat <<'EOF' > %q/shard-analysis.md
%sEOF
  cat <<'JSON'
{"meta":{"task_id":"task-artifact-restore","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"initial collect","changeset":[]}
JSON
  exit 0
fi
cat <<'EOF' > %q/shard-analysis.md
%sEOF
cat <<'JSON'
{"meta":{"task_id":"task-artifact-restore","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"retry stayed skeletal","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, validRetrySkeletalManifestJSON(), writeRoot, initialDoc, writeRoot, retryDoc)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write restore command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	_, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-artifact-restore",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatalf("expected failed artifact repair to reject skeletal collect artifacts")
	}
	code, message, ok := acpruntime.ClassifyError(err)
	if !ok || code != string(acpruntime.ErrorCodeRunnerParseFailed) {
		t.Fatalf("expected runner_parse_failed classification, got code=%q err=%v", code, err)
	}
	if !strings.Contains(message, "raw_output=reports/taskruns/raw/") {
		t.Fatalf("expected raw output diagnostics in message, got %q", message)
	}
	rawMetaFiles, globErr := filepath.Glob(filepath.Join(workspace, "reports", "taskruns", "raw", "*-meta.json"))
	if globErr != nil {
		t.Fatalf("glob raw output meta files: %v", globErr)
	}
	if len(rawMetaFiles) == 0 {
		t.Fatalf("expected raw output metadata under %s", filepath.Join(workspace, "reports", "taskruns", "raw"))
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	docContent := string(mustReadFile(t, filepath.Join(writeRoot, "shard-analysis.md")))
	if docContent != initialDoc {
		t.Fatalf("expected rollback to restore original doc, got %q", docContent)
	}
}

func TestHeadlessRunnerRepairsLegacyStyleManifestAfterSchemaValidRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "qwen-legacy-repair-count.txt")
	commandPath := filepath.Join(root, "qwen-legacy-manifest-repair.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
if [ "$count" -eq 1 ]; then
  cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
  cat <<'EOF' > %q/services.md
# Services
EOF
  cat <<'JSON'
{"meta":{"task_id":"task-legacy-manifest-repair","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"initial collect","changeset":[]}
JSON
  exit 0
fi
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'EOF' > %q/iac-overview.md
# IAC Overview
EOF
cat <<'JSON'
{"meta":{"task_id":"task-legacy-manifest-repair","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"artifact repair succeeded","changeset":[]}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, invalidLegacyStyleManifestJSON(), writeRoot, writeRoot, validRetryRichManifestJSON(), writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write repair command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-legacy-manifest-repair",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs",
		RepoScopes:   []string{"bank-of-anthos"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected legacy manifest repair to succeed: %v", err)
	}
	if result.TaskResult.Summary != "artifact repair succeeded" {
		t.Fatalf("expected repaired result summary, got %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "2" {
		t.Fatalf("expected exactly 2 runner invocations, got %q", count)
	}
	assessment, assessErr := assessRetryManifestAtWriteRoot(writeRoot)
	if assessErr != nil {
		t.Fatalf("assess repaired manifest: %v", assessErr)
	}
	if !assessment.Rich {
		t.Fatalf("expected repaired legacy manifest to be rich, got %#v", assessment)
	}
}

func TestHeadlessRunnerCanonicalizesRichManifestMissingMetadataWithoutRepairRetry(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	repoPath := filepath.Join(root, "bank-of-anthos")
	writeRoot := filepath.Join(root, "write-root")
	for _, dir := range []string{workspace, repoPath, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	manifest := "version: 1\nrepos:\n  - name: bank-of-anthos\n    path: " + repoPath + "\n"
	if err := os.WriteFile(filepath.Join(workspace, "workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write workspace manifest: %v", err)
	}

	stateFile := filepath.Join(root, "qwen-metadata-normalize-count.txt")
	commandPath := filepath.Join(root, "qwen-manifest-metadata-normalize.sh")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
count=0
if [ -f %q ]; then
  count="$(cat %q)"
fi
count=$((count + 1))
printf '%%s' "$count" > %q
mkdir -p %q
cat <<'JSON' > %q/shard-pack-manifest.json
%s
JSON
cat <<'EOF' > %q/service-catalog.md
# Service Catalog
EOF
cat <<'JSON'
{"meta":{"task_id":"task-manifest-metadata-normalize","step_id":"refresh.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"metadata normalized","changeset":[],"coverage":{"observed":["services"],"missing":["owner mappings"],"notes":["repo evidence preserved"]}}
JSON
`, stateFile, stateFile, stateFile, writeRoot, writeRoot, validRetryRichManifestMissingMetadataJSON(), writeRoot)
	if err := os.WriteFile(commandPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write metadata normalize command: %v", err)
	}

	runner := HeadlessRunner{Command: commandPath}
	result, err := runner.Run(context.Background(), acpruntime.Task{
		TaskID:       "task-manifest-metadata-normalize",
		RunID:        "run-1",
		StepID:       "refresh.step1.collect",
		ShardID:      "bank-of-anthos-docs",
		DomainID:     "bank-of-anthos",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		ArtifactRoot: "reports/taskruns/run-1/staging/shards/bank-of-anthos-docs",
		RepoScopes:   []string{"bank-of-anthos"},
		PathScopes:   []string{"docs"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("expected metadata canonicalization to succeed without repair retry: %v", err)
	}
	if result.TaskResult.Summary != "metadata normalized" {
		t.Fatalf("unexpected result summary %q", result.TaskResult.Summary)
	}
	if count := strings.TrimSpace(string(mustReadFile(t, stateFile))); count != "1" {
		t.Fatalf("expected exactly 1 runner invocation, got %q", count)
	}
	raw := mustReadFile(t, filepath.Join(writeRoot, "shard-pack-manifest.json"))
	assessment, assessErr := assessRetryManifestAtWriteRoot(writeRoot)
	if assessErr != nil {
		t.Fatalf("assess canonicalized manifest: %v", assessErr)
	}
	if !assessment.Rich {
		t.Fatalf("expected canonicalized manifest to stay rich, got %#v", assessment)
	}
	text := string(raw)
	for _, expected := range []string{`"run_id": "run-1"`, `"step_id": "refresh.step1.collect"`, `"shard_id": "bank-of-anthos-docs"`, `"agent_role": "shard-analyst"`, `"compatibility": {`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected canonicalized manifest to contain %s, got %q", expected, text)
		}
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

func TestRunQwenCommandPrefersWorkspaceAsCommandDirOverWriteRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	writeRoot := filepath.Join(workspace, "reports", "taskruns", "staging", "shards", "bank-of-anthos-docs")
	for _, dir := range []string{workspace, writeRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	task := acpruntime.Task{
		TaskID:       "task-dir-check",
		RunID:        "run-1",
		StepID:       "init.step1.collect",
		Workspace:    workspace,
		WriteRoot:    writeRoot,
		RepoScopes:   []string{"payments-service"},
		StartedAtUTC: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	validJSON := fmt.Sprintf(`{"meta":{"task_id":"%s","step_id":"%s","run_id":"%s","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z","workspace":"%s"},"summary":"ok","changeset":[{"op":"upsert_entity","entity":{"id":"svc.payments-service","type":"service","name":"Payments Service","attributes":{"repo_scope":"payments-service"},"provenance":{"kind":"observation","confidence":0.7,"evidence":[{"repo":"payments-service","path":"README.md"}]}}}]}`,
		task.TaskID,
		task.StepID,
		task.RunID,
		task.Workspace,
	)

	result, parseStage, parseErr, runErr := runQwenCommand(
		context.Background(),
		task,
		"sh",
		[]string{"-c", fmt.Sprintf("pwd >&2; printf '%%s' '%s'", validJSON)},
		runQwenOptions{},
	)
	if runErr != nil {
		t.Fatalf("unexpected runner error: %v", runErr)
	}
	if parseErr != nil {
		t.Fatalf("unexpected parse error at stage %q: %v", parseStage, parseErr)
	}
	if got := strings.TrimSpace(result.Stderr); got != workspace {
		t.Fatalf("expected qwen cwd to be workspace %q, got %q", workspace, got)
	}
}
