package orchestrator

import (
	"testing"
	"time"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestLogRuntimeOutputLeavesModelTelemetryAsPlainDiagnostics(t *testing.T) {
	t.Parallel()

	logs := []RunLogEntry{}
	execution := &pipelineExecution{
		pipelineRunProgressState: pipelineRunProgressState{
			onLog: func(entry RunLogEntry) {
				logs = append(logs, entry)
			},
		},
	}
	execution.clock = func() time.Time {
		return time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	}

	execution.logRuntimeOutput("init.step1.collect", "domain-a", acpruntime.ProviderClaudeCode, acpruntime.OutputChunk{
		Stream: acpruntime.OutputStreamStdout,
		Text:   `{"type":"result","modelUsage":{"kimi-for-coding":{"inputTokens":1}}}` + "\n",
	})

	if len(logs) != 1 {
		t.Fatalf("expected one runtime log entry, got %d", len(logs))
	}
	if got := logs[0].Level; got != RunLogLevelInfo {
		t.Fatalf("model telemetry should remain info-level diagnostics, got %s", got)
	}
	if len(logs[0].Fields) != 0 {
		t.Fatalf("model telemetry must not create structured attribution fields, got %#v", logs[0].Fields)
	}
	if len(execution.warnings) != 0 {
		t.Fatalf("model telemetry must not add runtime warnings, got %v", execution.warnings)
	}
}
