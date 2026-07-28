package contracts

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/validation"
)

type SourceQAAnswerCitation struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type SourceQAAnswer struct {
	Version           int                      `json:"version"`
	SourceRunID       string                   `json:"source_run_id"`
	AnswerDigest      string                   `json:"answer_digest"`
	ProposalTitle     string                   `json:"proposal_title"`
	Question          string                   `json:"question"`
	AnswerGeneratedAt string                   `json:"answer_generated_at"`
	Citations         []SourceQAAnswerCitation `json:"citations"`
	Unresolved        []string                 `json:"unresolved"`
	OperatorNote      string                   `json:"operator_note,omitempty"`
	CreatedAt         string                   `json:"created_at"`
}

func ParseSourceQAAnswer(raw []byte) (SourceQAAnswer, error) {
	if err := validation.ValidateRawJSON(validation.SourceQAAnswerSchema, raw); err != nil {
		return SourceQAAnswer{}, fmt.Errorf("source qa answer is invalid: %w", err)
	}
	var source SourceQAAnswer
	if err := json.Unmarshal(raw, &source); err != nil {
		return SourceQAAnswer{}, fmt.Errorf("decode source qa answer: %w", err)
	}
	problems := []string{}
	for label, value := range map[string]string{
		"source_run_id":       source.SourceRunID,
		"proposal_title":      source.ProposalTitle,
		"question":            source.Question,
		"answer_digest":       source.AnswerDigest,
		"created_at":          source.CreatedAt,
		"answer_generated_at": source.AnswerGeneratedAt,
	} {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, label+" is required")
		}
	}
	for _, value := range []struct {
		label string
		raw   string
	}{{"created_at", source.CreatedAt}, {"answer_generated_at", source.AnswerGeneratedAt}} {
		if _, err := time.Parse(time.RFC3339, strings.TrimSpace(value.raw)); err != nil {
			problems = append(problems, value.label+" must be RFC3339")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return SourceQAAnswer{}, fmt.Errorf("source qa answer is invalid: %s", strings.Join(problems, "; "))
	}
	return source, nil
}
