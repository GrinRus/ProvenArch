package runtime

import (
	"errors"
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
