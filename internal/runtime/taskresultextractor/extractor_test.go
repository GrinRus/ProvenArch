package taskresultextractor

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const validTaskResultJSON = `{"meta":{"task_id":"task-1","step_id":"init.step1.collect","runtime":{"name":"qwen-code","version":"test"},"started_at":"2026-04-03T12:00:00Z"},"summary":"ok","changeset":[]}`

func TestExtractFromRawTaskResultObject(t *testing.T) {
	t.Parallel()

	parsed, err := Extract([]byte(validTaskResultJSON))
	if err != nil {
		t.Fatalf("extract raw taskresult: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must remain valid taskresult: %v", err)
	}
}

func TestExtractFromEnvelopeResultString(t *testing.T) {
	t.Parallel()

	input := `{"type":"result","result":` + strconv.Quote(validTaskResultJSON) + `}`
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract envelope result: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must be valid taskresult: %v", err)
	}
}

func TestExtractFromJSONArrayEvents(t *testing.T) {
	t.Parallel()

	input := `[{"type":"system","subtype":"session_start"},{"type":"assistant","message":{"content":[{"type":"text","text":` + strconv.Quote(validTaskResultJSON) + `}]}}]`
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract from json events: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must be valid taskresult: %v", err)
	}
}

func TestExtractFromFencedText(t *testing.T) {
	t.Parallel()

	input := "```json\n" + validTaskResultJSON + "\n```"
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract fenced json: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must be valid taskresult: %v", err)
	}
}

func TestExtractFromNDJSONDataStreamWithPrefixNoise(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"event: message",
		"data: {\"type\":\"log\",\"message\":\"warming up\"}",
		"data: {\"type\":\"result\",\"result\":" + strconv.Quote(validTaskResultJSON) + "}",
	}, "\n")
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract ndjson stream: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must be valid taskresult: %v", err)
	}
}

func TestExtractFromNoisyOutputWithANSIAndControlChars(t *testing.T) {
	t.Parallel()

	input := "\x1b[32mINFO\x1b[0m runner output follows\n\x00" + validTaskResultJSON + "\x00"
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract noisy output: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err != nil {
		t.Fatalf("parsed output must be valid taskresult: %v", err)
	}
}

func TestExtractReturnsErrorForInvalidOutput(t *testing.T) {
	t.Parallel()

	if _, err := Extract([]byte("this is not taskresult")); err == nil {
		t.Fatalf("expected extraction error for invalid output")
	} else if !strings.Contains(err.Error(), "unable to extract valid TaskResult JSON") {
		t.Fatalf("expected top-level extraction context in error, got %v", err)
	}
}

func TestExtractReturnsSpecificErrorForEmptyEnvelopeResult(t *testing.T) {
	t.Parallel()

	_, err := Extract([]byte(`{"type":"result","result":""}`))
	if err == nil {
		t.Fatalf("expected extraction error for empty envelope result")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "envelope result is empty") {
		t.Fatalf("expected envelope-empty reason, got: %v", err)
	}
}
