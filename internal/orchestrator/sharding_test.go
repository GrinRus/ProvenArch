package orchestrator

import (
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/codexcode"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
)

func TestRuntimeMetaForRunnerCoversReleaseProviders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		runner      acpruntime.Runner
		wantName    string
		wantVersion string
	}{
		{name: "fake", runner: claudecode.FakeRunner{}, wantName: "claude-code", wantVersion: "fake"},
		{name: "claude", runner: claudecode.HeadlessRunner{}, wantName: "claude-code", wantVersion: "headless"},
		{name: "qwen", runner: qwencode.HeadlessRunner{}, wantName: "qwen-code", wantVersion: "headless"},
		{name: "codex", runner: codexcode.HeadlessRunner{}, wantName: "codex-code", wantVersion: "headless"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := runtimeMetaForRunner(tc.runner)
			if meta.Name != tc.wantName || meta.Version != tc.wantVersion {
				t.Fatalf("unexpected runtime meta: got=%+v want name=%q version=%q", meta, tc.wantName, tc.wantVersion)
			}
		})
	}
}
