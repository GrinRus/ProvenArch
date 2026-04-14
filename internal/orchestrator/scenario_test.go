package orchestrator

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const (
	envUpdateScenarioGolden = "ACP_UPDATE_SCENARIO_GOLDEN"
	envExportScenarioGolden = "ACP_EXPORT_SCENARIO_GOLDEN"
)

func TestScenarioFixturesDeterministicInitPipeline(t *testing.T) {
	t.Parallel()

	scenarios := []string{
		"single-service-http-postgres-gitlabci",
		"two-services-http-call-and-queue",
		"missing-owner-and-missing-cicd",
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			sourceRoot := filepath.Join("..", "..", "fixtures", "scenarios", scenario)
			firstSnapshot := runScenarioAndSnapshot(t, sourceRoot, true)
			secondSnapshot := runScenarioAndSnapshot(t, sourceRoot, false)

			if len(firstSnapshot) == 0 {
				t.Fatalf("expected deterministic snapshot files")
			}
			if len(firstSnapshot) != len(secondSnapshot) {
				t.Fatalf("snapshot file count mismatch: %d vs %d", len(firstSnapshot), len(secondSnapshot))
			}
			for path, firstHash := range firstSnapshot {
				secondHash, ok := secondSnapshot[path]
				if !ok {
					t.Fatalf("missing deterministic output %q in second snapshot", path)
				}
				if firstHash != secondHash {
					t.Fatalf("non-deterministic output for %q", path)
				}
			}
			assertSnapshotMatchesGolden(t, sourceRoot, firstSnapshot)
		})
	}
}

func runScenarioAndSnapshot(t *testing.T, sourceRoot string, exportReadable bool) map[string][32]byte {
	t.Helper()

	workspaceRoot := copyScenarioRoot(t, sourceRoot)

	ws, err := workspace.Open(filepath.Join(workspaceRoot, "workspace"))
	if err != nil {
		t.Fatalf("open workspace: %v", err)
	}

	runner := claudecode.RecordedRunner{
		ByStep: map[string]string{
			"init.step2.service_collect":  filepath.Join(workspaceRoot, "runner", "init.step2.service_collect.json"),
			"init.step4.service_findings": filepath.Join(workspaceRoot, "runner", "init.step4.service_findings.json"),
			"init.step5.global_review":    filepath.Join(workspaceRoot, "runner", "init.step5.global_review.json"),
		},
	}
	fixedClock := func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}
	service := NewService(
		WithRunner(runner),
		WithClock(fixedClock),
	)

	info, artifacts, err := service.Run(context.Background(), RunRequest{
		Workspace:      ws,
		Pipeline:       PipelineInit,
		NonInteractive: true,
	})
	if err != nil {
		t.Fatalf("run init pipeline: %v", err)
	}
	if info.Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded status, got %s (%s)", info.Status, info.Error)
	}
	if len(artifacts) == 0 {
		t.Fatalf("expected artifacts for scenario fixture at %s", sourceRoot)
	}

	required := []string{
		"reports/as-is/overview.md",
		"reports/diagrams/c4-context.mmd",
		"reports/findings/findings.md",
		"reports/coverage/summary.md",
		"reports/coverage/open-questions.md",
	}
	for _, rel := range required {
		abs := filepath.Join(ws.Path, rel)
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("expected artifact %q: %v", rel, err)
		}
	}

	snapshot := snapshotDeterministicOutputs(t, ws.Path)
	if exportReadable {
		maybeExportHumanReadableGolden(t, sourceRoot, ws.Path, snapshot)
	}
	return snapshot
}

func snapshotDeterministicOutputs(t *testing.T, workspacePath string) map[string][32]byte {
	t.Helper()

	includeRoots := []string{
		"charter",
		"model",
		"reports/as-is",
		"reports/diagrams",
		"reports/findings",
		"reports/coverage",
		"reports/agent-outputs",
		"proposals",
	}
	files := []string{}
	for _, rel := range includeRoots {
		abs := filepath.Join(workspacePath, rel)
		if err := filepath.WalkDir(abs, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			relPath, relErr := filepath.Rel(workspacePath, path)
			if relErr != nil {
				return relErr
			}
			if strings.HasPrefix(relPath, "reports/changelog") || strings.HasPrefix(relPath, "reports/taskruns") {
				return nil
			}
			files = append(files, filepath.ToSlash(relPath))
			return nil
		}); err != nil {
			t.Fatalf("walk deterministic output root %q: %v", rel, err)
		}
	}
	sort.Strings(files)

	snapshot := map[string][32]byte{}
	for _, rel := range files {
		content, err := os.ReadFile(filepath.Join(workspacePath, rel))
		if err != nil {
			t.Fatalf("read snapshot file %q: %v", rel, err)
		}
		snapshot[rel] = sha256.Sum256(content)
	}
	return snapshot
}

func TestDeterministicSnapshotScopeExcludesRunSpecificArtifacts(t *testing.T) {
	t.Parallel()

	scenario := "single-service-http-postgres-gitlabci"
	sourceRoot := filepath.Join("..", "..", "fixtures", "scenarios", scenario)
	snapshot := runScenarioAndSnapshot(t, sourceRoot, false)
	for path := range snapshot {
		if strings.HasPrefix(path, "reports/taskruns/") {
			t.Fatalf("run-specific artifact leaked into deterministic snapshot: %s", path)
		}
		if strings.HasPrefix(path, "reports/changelog/") {
			t.Fatalf("run-specific artifact leaked into deterministic snapshot: %s", path)
		}
	}
}

func copyScenarioRoot(t *testing.T, sourceRoot string) string {
	t.Helper()

	targetRoot := filepath.Join(t.TempDir(), "scenario")
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatalf("create target root: %v", err)
	}

	if err := filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(sourceRoot, path)
		if relErr != nil {
			return relErr
		}
		targetPath := filepath.Join(targetRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		sourceFile, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		targetFile, createErr := os.Create(targetPath)
		if createErr != nil {
			_ = sourceFile.Close()
			return createErr
		}
		if _, copyErr := io.Copy(targetFile, sourceFile); copyErr != nil {
			_ = targetFile.Close()
			_ = sourceFile.Close()
			return copyErr
		}
		if closeErr := targetFile.Close(); closeErr != nil {
			_ = sourceFile.Close()
			return closeErr
		}
		if closeErr := sourceFile.Close(); closeErr != nil {
			return closeErr
		}
		return nil
	}); err != nil {
		t.Fatalf("copy scenario root: %v", err)
	}

	return targetRoot
}

func TestScenarioFixtureLayoutExists(t *testing.T) {
	t.Parallel()

	scenarios := []string{
		"single-service-http-postgres-gitlabci",
		"two-services-http-call-and-queue",
		"missing-owner-and-missing-cicd",
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			root := filepath.Join("..", "..", "fixtures", "scenarios", scenario)
			for _, rel := range []string{
				"workspace/workspace.yaml",
				"runner/init.step2.service_collect.json",
				"runner/init.step4.service_findings.json",
				"runner/init.step5.global_review.json",
				"golden/README.md",
				"golden/snapshot.sha256",
			} {
				abs := filepath.Join(root, rel)
				if _, err := os.Stat(abs); err != nil {
					t.Fatalf("missing fixture file %s: %v", abs, err)
				}
			}
		})
	}
}

func TestScenarioRunnerFixturesContractAndSemantics(t *testing.T) {
	t.Parallel()

	scenarios := []string{
		"single-service-http-postgres-gitlabci",
		"two-services-http-call-and-queue",
		"missing-owner-and-missing-cicd",
	}
	stepExpectations := map[string]string{
		"runner/init.step2.service_collect.json":  "init.step2.service_collect",
		"runner/init.step4.service_findings.json": "init.step4.service_findings",
		"runner/init.step5.global_review.json":    "init.step5.global_review",
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			root := filepath.Join("..", "..", "fixtures", "scenarios", scenario)
			workspacePath, err := filepath.Abs(filepath.Join(root, "workspace"))
			if err != nil {
				t.Fatalf("resolve scenario workspace path: %v", err)
			}
			ws, err := workspace.Open(workspacePath)
			if err != nil {
				t.Fatalf("open scenario workspace fixture: %v", err)
			}
			expectedScopes := collectRepoScopes(ws.Manifest.Repos)

			for rel, expectedStepID := range stepExpectations {
				raw, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil {
					t.Fatalf("read runner fixture %s: %v", rel, err)
				}
				parsed, err := contracts.ParseTaskResult(raw)
				if err != nil {
					t.Fatalf("parse runner fixture %s: %v", rel, err)
				}
				if parsed.Meta.StepID != expectedStepID {
					t.Fatalf("fixture %s has unexpected meta.step_id=%q want=%q", rel, parsed.Meta.StepID, expectedStepID)
				}

				gotScopes := append([]string(nil), parsed.Meta.RepoScopes...)
				sort.Strings(gotScopes)
				if !equalStringSlices(gotScopes, expectedScopes) {
					t.Fatalf("fixture %s has unexpected repo_scopes=%v want=%v", rel, gotScopes, expectedScopes)
				}
			}
		})
	}
}

func TestScenarioDomainTaskEnvelopesDeterministic(t *testing.T) {
	t.Parallel()

	scenarios := []string{
		"single-service-http-postgres-gitlabci",
		"two-services-http-call-and-queue",
		"missing-owner-and-missing-cicd",
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			workspaceRoot := copyScenarioRoot(t, filepath.Join("..", "..", "fixtures", "scenarios", scenario))
			ws, err := workspace.Open(filepath.Join(workspaceRoot, "workspace"))
			if err != nil {
				t.Fatalf("open workspace: %v", err)
			}

			runner := claudecode.RecordedRunner{
				ByStep: map[string]string{
					"init.step2.service_collect":  filepath.Join(workspaceRoot, "runner", "init.step2.service_collect.json"),
					"init.step4.service_findings": filepath.Join(workspaceRoot, "runner", "init.step4.service_findings.json"),
					"init.step5.global_review":    filepath.Join(workspaceRoot, "runner", "init.step5.global_review.json"),
				},
			}
			service := NewService(
				WithRunner(runner),
				WithClock(func() time.Time { return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC) }),
			)
			runInfo, _, err := service.Run(context.Background(), RunRequest{
				Workspace:      ws,
				Pipeline:       PipelineInit,
				NonInteractive: true,
			})
			if err != nil {
				t.Fatalf("run scenario for domain envelope validation: %v", err)
			}
			if runInfo.Status != RunStatusSucceeded {
				t.Fatalf("expected succeeded status, got %s (%s)", runInfo.Status, runInfo.Error)
			}

			domainIDs, err := loadCanonicalDomainIDs(ws)
			if err != nil {
				t.Fatalf("load canonical domain ids: %v", err)
			}
			for _, domainID := range domainIDs {
				path := filepath.Join(ws.Path, "reports", "agent-outputs", "domains", fmt.Sprintf("%s.task-envelope.json", sanitizeDomainArtifactSlug(domainID)))
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read domain task envelope %s: %v", path, err)
				}
				var payload map[string]any
				if err := json.Unmarshal(content, &payload); err != nil {
					t.Fatalf("decode domain task envelope %s: %v", path, err)
				}
				raw := string(content)
				if strings.Contains(raw, "run_") {
					t.Fatalf("domain task envelope must be deterministic and must not contain run id: %s", path)
				}
				if strings.Contains(raw, "2026-") || strings.Contains(raw, "T12:") {
					t.Fatalf("domain task envelope must be deterministic and must not contain timestamps: %s", path)
				}
			}

			taskrunPaths, err := filepath.Glob(filepath.Join(ws.Path, "reports", "taskruns", "*-init-step2-service_collect-domain-service-*.json"))
			if err != nil {
				t.Fatalf("glob per-domain taskruns: %v", err)
			}
			if len(taskrunPaths) == 0 {
				t.Fatalf("expected non-empty service taskruns for step2 collect")
			}
			for _, path := range taskrunPaths {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read service taskrun %s: %v", path, err)
				}
				var payload struct {
					Meta struct {
						StepID string `json:"step_id"`
					} `json:"meta"`
				}
				if err := json.Unmarshal(content, &payload); err != nil {
					t.Fatalf("decode service taskrun %s: %v", path, err)
				}
				if payload.Meta.StepID != "init.step2.service_collect" {
					t.Fatalf("expected service taskrun step id init.step2.service_collect, got %q", payload.Meta.StepID)
				}
			}
		})
	}
}

func assertSnapshotMatchesGolden(t *testing.T, sourceRoot string, snapshot map[string][32]byte) {
	t.Helper()

	goldenPath := filepath.Join(sourceRoot, "golden", "snapshot.sha256")
	if strings.TrimSpace(os.Getenv(envUpdateScenarioGolden)) == "1" {
		writeSnapshotGoldenFile(t, goldenPath, snapshot)
		return
	}
	file, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("open golden snapshot %q: %v", goldenPath, err)
	}
	defer file.Close()

	expected := map[string][32]byte{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Fatalf("invalid golden snapshot line %q", line)
		}
		hashText := strings.TrimSpace(fields[0])
		relPath := strings.TrimSpace(fields[len(fields)-1])
		hashBytes, err := hex.DecodeString(hashText)
		if err != nil {
			t.Fatalf("decode golden hash %q: %v", hashText, err)
		}
		if len(hashBytes) != sha256.Size {
			t.Fatalf("golden hash %q has unexpected length %d", hashText, len(hashBytes))
		}
		var hash [sha256.Size]byte
		copy(hash[:], hashBytes)
		expected[relPath] = hash
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan golden snapshot: %v", err)
	}

	if len(expected) != len(snapshot) {
		t.Fatalf("golden snapshot file count mismatch: expected=%d actual=%d", len(expected), len(snapshot))
	}
	for path, expectedHash := range expected {
		actualHash, ok := snapshot[path]
		if !ok {
			t.Fatalf("missing output %q from actual snapshot", path)
		}
		if actualHash != expectedHash {
			t.Fatalf("golden mismatch for %q", path)
		}
	}
}

func writeSnapshotGoldenFile(t *testing.T, goldenPath string, snapshot map[string][32]byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
		t.Fatalf("create golden dir for %q: %v", goldenPath, err)
	}
	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	builder := strings.Builder{}
	for _, path := range paths {
		hash := snapshot[path]
		builder.WriteString(hex.EncodeToString(hash[:]))
		builder.WriteString("  ")
		builder.WriteString(path)
		builder.WriteString("\n")
	}
	if err := os.WriteFile(goldenPath, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write golden snapshot %q: %v", goldenPath, err)
	}
}

func maybeExportHumanReadableGolden(t *testing.T, sourceRoot string, workspacePath string, snapshot map[string][32]byte) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(envExportScenarioGolden)) != "1" {
		return
	}
	targetRoot := filepath.Join(sourceRoot, "golden", "readable")
	if err := os.RemoveAll(targetRoot); err != nil {
		t.Fatalf("cleanup readable golden dir %q: %v", targetRoot, err)
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		t.Fatalf("create readable golden dir %q: %v", targetRoot, err)
	}

	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		content, err := os.ReadFile(filepath.Join(workspacePath, rel))
		if err != nil {
			t.Fatalf("read deterministic output %q for readable golden export: %v", rel, err)
		}
		target := filepath.Join(targetRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create readable golden subdir for %q: %v", target, err)
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			t.Fatalf("write readable golden file %q: %v", target, err)
		}
	}
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func Example_scenarioReadme() {
	fmt.Println("fixtures/scenarios contain synthetic workspaces with recorded runner artifacts")
	// Output: fixtures/scenarios contain synthetic workspaces with recorded runner artifacts
}
