package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/validation"
)

type TaskResult struct {
	Meta      Meta        `json:"meta"`
	Summary   string      `json:"summary"`
	Changeset []Operation `json:"changeset"`
	Questions []Question  `json:"questions,omitempty"`
	Coverage  *Coverage   `json:"coverage,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
	Debug     any         `json:"debug,omitempty"`
}

type Meta struct {
	TaskID     string      `json:"task_id"`
	StepID     string      `json:"step_id"`
	RunID      string      `json:"run_id,omitempty"`
	Runtime    RuntimeMeta `json:"runtime"`
	StartedAt  string      `json:"started_at"`
	FinishedAt string      `json:"finished_at,omitempty"`
	Workspace  string      `json:"workspace,omitempty"`
	ShardID    string      `json:"shard_id,omitempty"`
	RepoScope  string      `json:"repo_scope,omitempty"`
	RepoScopes []string    `json:"repo_scopes,omitempty"`
	PathScopes []string    `json:"path_scopes,omitempty"`
}

type RuntimeMeta struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Operation struct {
	Op          string       `json:"op"`
	Entity      *Entity      `json:"entity,omitempty"`
	EntityID    string       `json:"entity_id,omitempty"`
	Edge        *Edge        `json:"edge,omitempty"`
	EdgeID      string       `json:"edge_id,omitempty"`
	Finding     *Finding     `json:"finding,omitempty"`
	DocArtifact *DocArtifact `json:"doc_artifact,omitempty"`
	Question    *Question    `json:"question,omitempty"`
	Coverage    *Coverage    `json:"coverage,omitempty"`
	Reason      string       `json:"reason,omitempty"`
	TargetPath  string       `json:"target_path,omitempty"`
}

type Entity struct {
	ID          string     `json:"id" yaml:"id"`
	Type        string     `json:"type" yaml:"type"`
	Name        string     `json:"name" yaml:"name"`
	Aliases     []string   `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Tags        []string   `json:"tags,omitempty" yaml:"tags,omitempty"`
	Attributes  any        `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	OwnerTeamID string     `json:"owner_team_id,omitempty" yaml:"owner_team_id,omitempty"`
	Provenance  Provenance `json:"provenance" yaml:"provenance"`
}

type Edge struct {
	ID         string     `json:"id" yaml:"id"`
	Type       string     `json:"type" yaml:"type"`
	From       string     `json:"from" yaml:"from"`
	To         string     `json:"to" yaml:"to"`
	Name       string     `json:"name,omitempty" yaml:"name,omitempty"`
	Attributes any        `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Provenance Provenance `json:"provenance" yaml:"provenance"`
}

type Finding struct {
	ID          string     `json:"id"`
	Severity    string     `json:"severity"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	RuleID      string     `json:"rule_id,omitempty"`
	RelatedIDs  []string   `json:"related_ids,omitempty"`
	Provenance  Provenance `json:"provenance"`
}

type DocArtifact struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Format     string   `json:"format,omitempty"`
	RelatedIDs []string `json:"related_ids,omitempty"`
}

type Question struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	Priority   string   `json:"priority,omitempty"`
	RelatedIDs []string `json:"related_ids,omitempty"`
}

type Coverage struct {
	Observed []string `json:"observed,omitempty"`
	Missing  []string `json:"missing,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

type Provenance struct {
	Kind       string     `json:"kind" yaml:"kind"`
	Confidence float64    `json:"confidence" yaml:"confidence"`
	Evidence   []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type Evidence struct {
	Repo        string     `json:"repo" yaml:"repo"`
	Ref         string     `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path        string     `json:"path" yaml:"path"`
	Lines       *LineRange `json:"lines,omitempty" yaml:"lines,omitempty"`
	ExcerptHash string     `json:"excerpt_hash,omitempty" yaml:"excerpt_hash,omitempty"`
	Excerpt     string     `json:"excerpt,omitempty" yaml:"excerpt,omitempty"`
}

type LineRange struct {
	Start int `json:"start" yaml:"start"`
	End   int `json:"end" yaml:"end"`
}

var (
	ErrTaskResultRequired = errors.New("taskresult payload is required")
)

var supportedOps = map[string]struct{}{
	"upsert_entity":    {},
	"remove_entity":    {},
	"upsert_edge":      {},
	"remove_edge":      {},
	"add_finding":      {},
	"add_doc_artifact": {},
	"add_question":     {},
	"set_coverage":     {},
}

func ParseTaskResult(raw []byte) (TaskResult, error) {
	if len(raw) == 0 {
		return TaskResult{}, ErrTaskResultRequired
	}
	if err := validation.ValidateRawJSON(validation.TaskResultSchema, raw); err != nil {
		return TaskResult{}, fmt.Errorf("taskresult is invalid: %w", err)
	}

	var result TaskResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return TaskResult{}, fmt.Errorf("decode taskresult: %w", err)
	}
	if err := validateTaskResult(result); err != nil {
		return TaskResult{}, err
	}
	return result, nil
}

func NormalizeTaskResult(result TaskResult) TaskResult {
	normalized := result

	questionsByID := map[string]Question{}
	orderedQuestionIDs := make([]string, 0, len(result.Questions))
	for _, question := range result.Questions {
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		if _, exists := questionsByID[question.ID]; exists {
			continue
		}
		questionsByID[question.ID] = question
		orderedQuestionIDs = append(orderedQuestionIDs, question.ID)
	}

	coverage := normalized.Coverage
	if coverage == nil {
		coverage = &Coverage{}
	}

	filteredChangeset := make([]Operation, 0, len(result.Changeset))
	for _, op := range result.Changeset {
		switch op.Op {
		case "add_question":
			if op.Question == nil || strings.TrimSpace(op.Question.ID) == "" {
				continue
			}
			if _, exists := questionsByID[op.Question.ID]; exists {
				continue
			}
			questionsByID[op.Question.ID] = *op.Question
			orderedQuestionIDs = append(orderedQuestionIDs, op.Question.ID)
		case "set_coverage":
			if op.Coverage == nil {
				continue
			}
			coverage.Observed = uniquePreserveOrder(append(coverage.Observed, op.Coverage.Observed...))
			coverage.Missing = uniquePreserveOrder(append(coverage.Missing, op.Coverage.Missing...))
			coverage.Notes = uniquePreserveOrder(append(coverage.Notes, op.Coverage.Notes...))
		default:
			filteredChangeset = append(filteredChangeset, op)
		}
	}

	normalized.Changeset = filteredChangeset

	normalizedQuestions := make([]Question, 0, len(orderedQuestionIDs))
	for _, id := range orderedQuestionIDs {
		normalizedQuestions = append(normalizedQuestions, questionsByID[id])
	}
	normalized.Questions = normalizedQuestions

	coverage.Observed = uniquePreserveOrder(coverage.Observed)
	coverage.Missing = uniquePreserveOrder(coverage.Missing)
	coverage.Notes = uniquePreserveOrder(coverage.Notes)
	if len(coverage.Observed) == 0 && len(coverage.Missing) == 0 && len(coverage.Notes) == 0 {
		normalized.Coverage = nil
	} else {
		normalized.Coverage = coverage
	}

	return normalized
}

func validateTaskResult(result TaskResult) error {
	var problems []string

	if strings.TrimSpace(result.Meta.TaskID) == "" {
		problems = append(problems, "meta.task_id is required")
	}
	if strings.TrimSpace(result.Meta.StepID) == "" {
		problems = append(problems, "meta.step_id is required")
	}
	if strings.TrimSpace(result.Meta.Runtime.Name) == "" {
		problems = append(problems, "meta.runtime.name is required")
	}
	if strings.TrimSpace(result.Meta.StartedAt) == "" {
		problems = append(problems, "meta.started_at is required")
	}
	if strings.TrimSpace(result.Summary) == "" {
		problems = append(problems, "summary is required")
	}

	for idx, op := range result.Changeset {
		label := fmt.Sprintf("changeset[%d]", idx)
		if _, ok := supportedOps[op.Op]; !ok {
			problems = append(problems, fmt.Sprintf("%s has unsupported op %q", label, op.Op))
			continue
		}
		switch op.Op {
		case "upsert_entity":
			if op.Entity == nil {
				problems = append(problems, label+" requires entity")
				continue
			}
			if err := validateEntity(*op.Entity); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", label, err))
			}
		case "remove_entity":
			if strings.TrimSpace(op.EntityID) == "" {
				problems = append(problems, label+" requires entity_id")
			}
		case "upsert_edge":
			if op.Edge == nil {
				problems = append(problems, label+" requires edge")
				continue
			}
			if err := validateEdge(*op.Edge); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", label, err))
			}
		case "remove_edge":
			if strings.TrimSpace(op.EdgeID) == "" {
				problems = append(problems, label+" requires edge_id")
			}
		case "add_finding":
			if op.Finding == nil {
				problems = append(problems, label+" requires finding")
				continue
			}
			if err := validateFinding(*op.Finding); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", label, err))
			}
		case "add_doc_artifact":
			if op.DocArtifact == nil {
				problems = append(problems, label+" requires doc_artifact")
				continue
			}
			if strings.TrimSpace(op.DocArtifact.ID) == "" ||
				strings.TrimSpace(op.DocArtifact.Kind) == "" ||
				strings.TrimSpace(op.DocArtifact.Title) == "" ||
				strings.TrimSpace(op.DocArtifact.Path) == "" {
				problems = append(problems, label+" doc_artifact requires id, kind, title, path")
			}
		case "add_question":
			if op.Question == nil {
				problems = append(problems, label+" requires question")
				continue
			}
			if strings.TrimSpace(op.Question.ID) == "" || strings.TrimSpace(op.Question.Text) == "" {
				problems = append(problems, label+" question requires id and text")
			}
		case "set_coverage":
			if op.Coverage == nil {
				problems = append(problems, label+" requires coverage")
			}
		}
	}

	for idx, question := range result.Questions {
		if strings.TrimSpace(question.ID) == "" || strings.TrimSpace(question.Text) == "" {
			problems = append(problems, fmt.Sprintf("questions[%d] requires id and text", idx))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("taskresult is invalid: %s", strings.Join(problems, "; "))
}

func validateEntity(entity Entity) error {
	if strings.TrimSpace(entity.ID) == "" || strings.TrimSpace(entity.Type) == "" || strings.TrimSpace(entity.Name) == "" {
		return errors.New("entity requires id, type, name")
	}
	return validateProvenance(entity.Provenance)
}

func validateEdge(edge Edge) error {
	if strings.TrimSpace(edge.ID) == "" || strings.TrimSpace(edge.Type) == "" ||
		strings.TrimSpace(edge.From) == "" || strings.TrimSpace(edge.To) == "" {
		return errors.New("edge requires id, type, from, to")
	}
	return validateProvenance(edge.Provenance)
}

func validateFinding(finding Finding) error {
	if strings.TrimSpace(finding.ID) == "" || strings.TrimSpace(finding.Severity) == "" || strings.TrimSpace(finding.Title) == "" {
		return errors.New("finding requires id, severity, title")
	}
	return validateProvenance(finding.Provenance)
}

func validateProvenance(provenance Provenance) error {
	switch provenance.Kind {
	case "observation", "inference", "assertion":
	default:
		return fmt.Errorf("unsupported provenance.kind %q", provenance.Kind)
	}
	if provenance.Confidence < 0 || provenance.Confidence > 1 {
		return errors.New("provenance.confidence must be within [0,1]")
	}
	if provenance.Kind == "observation" && len(provenance.Evidence) == 0 {
		return errors.New("observation requires non-empty evidence")
	}
	for idx, evidence := range provenance.Evidence {
		if strings.TrimSpace(evidence.Repo) == "" || strings.TrimSpace(evidence.Path) == "" {
			return fmt.Errorf("evidence[%d] requires repo and path", idx)
		}
		if evidence.Lines != nil {
			if evidence.Lines.Start <= 0 || evidence.Lines.End <= 0 {
				return fmt.Errorf("evidence[%d].lines must be > 0", idx)
			}
		}
	}
	return nil
}

func uniquePreserveOrder(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
