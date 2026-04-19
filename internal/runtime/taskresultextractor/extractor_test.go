package taskresultextractor

import (
	"os"
	"path/filepath"
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

func TestExtractFromEnvelopeResultStringWithSummaryField(t *testing.T) {
	t.Parallel()

	input := `{"summary":"runner envelope","result":` + strconv.Quote(validTaskResultJSON) + `}`
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("extract envelope with summary field: %v", err)
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

func TestExtractReturnsErrorForCapturedQwenLiveInvalidStdoutFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "testdata", "qwen_live_bank_of_anthos_invalid_stdout.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live stdout fixture: %v", err)
	}

	_, err = Extract(raw)
	if err == nil {
		t.Fatalf("expected extraction error for captured live invalid stdout fixture")
	}
	if !strings.Contains(err.Error(), "unable to extract valid TaskResult JSON") {
		t.Fatalf("expected top-level extraction context in error, got %v", err)
	}
}

func TestExtractReturnsErrorForCapturedQwenLiveIACInvalidStdoutFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "testdata", "qwen_live_bank_of_anthos_iac_invalid_stdout.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live iac stdout fixture: %v", err)
	}

	_, err = Extract(raw)
	if err == nil {
		t.Fatalf("expected extraction error for captured live iac invalid stdout fixture")
	}
	if !strings.Contains(err.Error(), "unable to extract valid TaskResult JSON") {
		t.Fatalf("expected top-level extraction context in error, got %v", err)
	}
}

func TestExtractReturnsErrorForCapturedQwenLiveExtrasInvalidStdoutFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "testdata", "qwen_live_bank_of_anthos_extras_invalid_stdout.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read live extras stdout fixture: %v", err)
	}

	_, err = Extract(raw)
	if err == nil {
		t.Fatalf("expected extraction error for captured live extras invalid stdout fixture")
	}
	if !strings.Contains(err.Error(), "unable to extract valid TaskResult JSON") {
		t.Fatalf("expected top-level extraction context in error, got %v", err)
	}
}

func TestExtractReturnsCandidateFromCapturedQwenLegacyDocArtifactEventStreamFixture(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "testdata", "qwen_live_bank_extras_legacy_doc_artifact_stdout.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read legacy doc_artifact stdout fixture: %v", err)
	}

	parsed, err := Extract(raw)
	if err != nil {
		t.Fatalf("expected extractor to lift inner candidate from event-stream fixture: %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err == nil {
		t.Fatalf("expected schema validation to fail for legacy artifact payload")
	}
	if !strings.Contains(string(parsed), `"artifact"`) {
		t.Fatalf("expected extracted candidate to preserve legacy artifact payload, got %q", string(parsed))
	}
}

func TestExtractReturnsCandidateObjectEvenWhenSchemaInvalid(t *testing.T) {
	t.Parallel()

	input := `{"meta":{"task_id":"task-1"}}`
	parsed, err := Extract([]byte(input))
	if err != nil {
		t.Fatalf("expected extractor to return candidate object for schema validation stage, got %v", err)
	}
	if _, err := contracts.ParseTaskResult(parsed); err == nil {
		t.Fatalf("expected schema validation to fail for extracted candidate")
	}
}
