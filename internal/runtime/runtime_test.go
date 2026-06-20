package runtime

import (
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestWrapRunnerErrorWithDiagnosticsPreservesRawOutputRefs(t *testing.T) {
	t.Parallel()

	rawRefs := contracts.RuntimeOutputRefs{
		Stdout:   "reports/taskruns/raw/stdout.log",
		Stderr:   "reports/taskruns/raw/stderr.log",
		Metadata: "reports/taskruns/raw/meta.json",
	}

	err := WrapRunnerErrorWithDiagnostics(
		ProviderQwenCode,
		ErrorCodeRuntimeContract,
		"runtime contract failed",
		"stdout-text",
		"stderr-text",
		rawRefs,
		errors.New("boom"),
	)

	var runnerErr RunnerError
	if !errors.As(err, &runnerErr) {
		t.Fatalf("expected RunnerError, got %T", err)
	}
	if runnerErr.RawOutputRefs != rawRefs {
		t.Fatalf("expected raw output refs %#v, got %#v", rawRefs, runnerErr.RawOutputRefs)
	}
}

func TestCoreAORPackagesDoNotReferenceLiveReleaseEvidenceNames(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	forbidden := []string{
		"swe_" + "ux_assessment",
		"swe_" + "artifact_quality_assessment",
		"release_" + "verdict",
		"execution_" + "report",
	}
	scanRoots := []string{
		filepath.Join(repoRoot, "internal", "runtime"),
		filepath.Join(repoRoot, "internal", "orchestrator"),
		filepath.Join(repoRoot, "internal", "runtimedrafts"),
	}
	for _, root := range scanRoots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			for _, token := range forbidden {
				if strings.Contains(text, token) {
					t.Fatalf("core AOR production package %s must not reference live/manual release evidence name %q", path, token)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
}
