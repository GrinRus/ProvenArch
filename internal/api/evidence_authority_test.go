package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GrinRus/ProvenArch/internal/orchestrator"
)

func TestQARunPayloadUsesExactSnapshotAndAuditAuthorities(t *testing.T) {
	server := newTestServer(t)
	ws := server.getWorkspace()
	run := orchestrator.RunInfo{
		RunID:     "qa-selected",
		Pipeline:  string(orchestrator.PipelineQA),
		Status:    orchestrator.RunStatusCanceled,
		StartedAt: time.Now().UTC(),
	}

	payload, err := server.formatQARunPayload(ws, run)
	if err != nil {
		t.Fatalf("format canceled QA run: %v", err)
	}
	answerAuthority, ok := payload["answer_authority"].(evidenceAuthority)
	if !ok || answerAuthority.Mode != evidenceAuthorityQASnapshot || answerAuthority.RunID != run.RunID {
		t.Fatalf("unexpected answer authority: %#v", payload["answer_authority"])
	}
	auditAuthority, ok := payload["audit_authority"].(evidenceAuthority)
	if !ok || auditAuthority.Mode != evidenceAuthorityQAAudit || auditAuthority.RunID != run.RunID {
		t.Fatalf("unexpected audit authority: %#v", payload["audit_authority"])
	}
	if payload["answer_status"] != "not_produced" || payload["answer"] != nil {
		t.Fatalf("canceled run must not fall back to another answer: %#v", payload)
	}
}

func TestSucceededQARunWithoutOwnAnswerIsUnavailable(t *testing.T) {
	server := newTestServer(t)
	ws := server.getWorkspace()
	run := orchestrator.RunInfo{
		RunID:     "qa-missing-answer",
		Pipeline:  string(orchestrator.PipelineQA),
		Status:    orchestrator.RunStatusSucceeded,
		StartedAt: time.Now().UTC(),
	}
	otherAnswer := filepath.Join(ws.Path, "reports", "taskruns", "qa-other", "qa", "qa-answer.json")
	if err := os.MkdirAll(filepath.Dir(otherAnswer), 0o755); err != nil {
		t.Fatalf("create other answer parent: %v", err)
	}
	if err := os.WriteFile(otherAnswer, []byte(`{"answer":"wrong run"}`), 0o644); err != nil {
		t.Fatalf("write other answer: %v", err)
	}

	if _, err := server.formatQARunPayload(ws, run); err == nil {
		t.Fatal("succeeded run without its own answer must return qa_answer_unavailable")
	}
}
