package runnerdiag

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildExecFailurePrefersStderr(t *testing.T) {
	t.Parallel()

	err := BuildExecFailure(errors.New("exit status 1"), "stdout line", "stderr detail")
	if got := err.Error(); got != "stderr detail" {
		t.Fatalf("unexpected stderr-priority error: %q", got)
	}
}

func TestBuildExecFailureUsesStdoutExcerptWhenStderrEmpty(t *testing.T) {
	t.Parallel()

	err := BuildExecFailure(errors.New("exit status 1"), "line one\nline two", "")
	got := err.Error()
	if !strings.Contains(got, "stdout_excerpt=") {
		t.Fatalf("expected stdout excerpt in error, got %q", got)
	}
	if !strings.Contains(got, "line one line two") {
		t.Fatalf("expected normalized stdout snippet, got %q", got)
	}
}
