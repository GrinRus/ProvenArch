package qwencode

import (
	"strings"
	"testing"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func TestForwardStreamOutputEnforcesHardCapAndTruncates(t *testing.T) {
	t.Parallel()

	var chunks []acpruntime.OutputChunk
	task := acpruntime.Task{
		OnOutput: func(chunk acpruntime.OutputChunk) {
			chunks = append(chunks, chunk)
		},
	}
	budget := &streamedOutputBudget{
		forwardedBytes: acpruntime.RuntimeOutputStreamHardCapBytes - 3,
	}

	forwardStreamOutput(task, acpruntime.OutputStreamStderr, "abcdef\nanother-line\n", budget)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (partial line + truncation marker), got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Truncated {
		t.Fatalf("first chunk must be regular output, got %+v", chunks[0])
	}
	if chunks[0].Text != "abc" {
		t.Fatalf("expected first chunk to be clipped to remaining cap bytes, got %q", chunks[0].Text)
	}
	if chunks[1].Stream != acpruntime.OutputStreamStderr || !chunks[1].Truncated {
		t.Fatalf("expected second chunk to be truncation marker, got %+v", chunks[1])
	}
	if budget.truncated != true {
		t.Fatalf("expected budget.truncated=true after cap hit")
	}
}

func TestCaptureCommandStreamForwardsCRLFAndFinalLineWithoutNewline(t *testing.T) {
	t.Parallel()

	input := "stderr line one\r\nstderr line two without trailing newline"
	sink := &commandOutputBuffer{}
	var chunks []acpruntime.OutputChunk

	task := acpruntime.Task{
		OnOutput: func(chunk acpruntime.OutputChunk) {
			chunks = append(chunks, chunk)
		},
	}

	if err := captureCommandStream(strings.NewReader(input), sink, task, acpruntime.OutputStreamStderr); err != nil {
		t.Fatalf("capture command stream: %v", err)
	}
	if sink.String() != input {
		t.Fatalf("expected sink to preserve raw stream bytes, got %q", sink.String())
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 forwarded chunks, got %d (%+v)", len(chunks), chunks)
	}
	if chunks[0].Stream != acpruntime.OutputStreamStderr || chunks[0].Text != "stderr line one" || chunks[0].Truncated {
		t.Fatalf("unexpected first chunk: %+v", chunks[0])
	}
	if chunks[1].Stream != acpruntime.OutputStreamStderr || chunks[1].Text != "stderr line two without trailing newline" || chunks[1].Truncated {
		t.Fatalf("unexpected second chunk: %+v", chunks[1])
	}
}
