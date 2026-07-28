package proposaldraft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	"github.com/GrinRus/ProvenArch/internal/qa"
	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

var (
	ErrStaleDigest        = errors.New("qa answer digest is stale")
	ErrAlreadyExists      = errors.New("proposal draft already exists")
	ErrInvalidSlug        = errors.New("proposal slug is invalid")
	ErrUnresolvedCitation = errors.New("qa answer citation is unresolved")
)

var proposalSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Input struct {
	RunID                string
	Title                string
	Slug                 string
	OperatorNote         string
	ExpectedAnswerDigest string
	AnswerRaw            []byte
	ContextRaw           []byte
	CreatedAt            time.Time
}

type Result struct {
	Path         string `json:"path"`
	ProposalPath string `json:"proposal_path"`
	EvidencePath string `json:"evidence_path"`
	SourcePath   string `json:"source_path"`
	AnswerDigest string `json:"answer_digest"`
}

func AnswerDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func Create(ws workspace.Root, input Input) (Result, error) {
	runID := strings.TrimSpace(input.RunID)
	title := strings.TrimSpace(input.Title)
	if runID == "" || title == "" {
		return Result{}, fmt.Errorf("run id and title are required")
	}
	answer, err := qa.ParseAnswer(input.AnswerRaw)
	if err != nil {
		return Result{}, err
	}
	contextPack, err := qa.ParseContextPack(input.ContextRaw)
	if err != nil {
		return Result{}, err
	}
	if answer.RunID != runID || contextPack.RunID != runID {
		return Result{}, fmt.Errorf("%w: answer/context run identity does not match %q", ErrUnresolvedCitation, runID)
	}
	if err := qa.ValidateAnswerAgainstContext(answer, contextPack); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrUnresolvedCitation, err)
	}
	digest := AnswerDigest(input.AnswerRaw)
	if !strings.EqualFold(strings.TrimSpace(input.ExpectedAnswerDigest), digest) {
		return Result{}, ErrStaleDigest
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = slugutil.Slugify(title)
	}
	if !proposalSlugPattern.MatchString(slug) || slug != slugutil.Slugify(slug) {
		return Result{}, ErrInvalidSlug
	}
	runSlug := slugutil.Slugify(runID)
	if runSlug == "" {
		return Result{}, ErrInvalidSlug
	}
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	createdAt = createdAt.UTC()
	packageRel := path.Join("proposals", "qa-synthesis-"+runSlug+"-"+slug)
	citations := make([]contracts.SourceQAAnswerCitation, 0, len(answer.Citations))
	for _, citation := range answer.Citations {
		citations = append(citations, contracts.SourceQAAnswerCitation{Path: citation.Path, Reason: citation.Reason})
	}
	sort.Slice(citations, func(i, j int) bool {
		if citations[i].Path == citations[j].Path {
			return citations[i].Reason < citations[j].Reason
		}
		return citations[i].Path < citations[j].Path
	})
	unresolved := append([]string{}, answer.Unresolved...)
	source := contracts.SourceQAAnswer{
		Version:           1,
		SourceRunID:       runID,
		AnswerDigest:      digest,
		ProposalTitle:     title,
		Question:          answer.Question,
		AnswerGeneratedAt: answer.GeneratedAt,
		Citations:         citations,
		Unresolved:        unresolved,
		OperatorNote:      strings.TrimSpace(input.OperatorNote),
		CreatedAt:         createdAt.Format(time.RFC3339),
	}
	sourceRaw, err := json.MarshalIndent(source, "", "  ")
	if err != nil {
		return Result{}, err
	}
	sourceRaw = append(sourceRaw, '\n')
	if _, err := contracts.ParseSourceQAAnswer(sourceRaw); err != nil {
		return Result{}, err
	}

	files := map[string][]byte{
		"proposal.md":           []byte(renderProposal(title, answer, input.OperatorNote)),
		"evidence.md":           []byte(renderEvidence(answer)),
		"source-qa-answer.json": sourceRaw,
	}
	if err := ws.WriteDirectoryAtomicExclusive(packageRel, files); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return Result{}, ErrAlreadyExists
		}
		return Result{}, err
	}
	return Result{
		Path:         packageRel,
		ProposalPath: path.Join(packageRel, "proposal.md"),
		EvidencePath: path.Join(packageRel, "evidence.md"),
		SourcePath:   path.Join(packageRel, "source-qa-answer.json"),
		AnswerDigest: digest,
	}, nil
}

func renderProposal(title string, answer qa.Answer, operatorNote string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# %s\n\n", title)
	builder.WriteString("## Decision / recommended operator action\n\n")
	builder.WriteString(strings.TrimSpace(answer.Answer))
	builder.WriteString("\n\n## Evidence\n\nSee [evidence.md](./evidence.md) and `source-qa-answer.json`.\n\n")
	builder.WriteString("## Citations\n\n")
	if len(answer.Citations) == 0 {
		builder.WriteString("- No resolved citations; review is required.\n")
	} else {
		for _, citation := range answer.Citations {
			fmt.Fprintf(&builder, "- `%s` — %s\n", citation.Path, citation.Reason)
		}
	}
	builder.WriteString("\n## Unresolved\n\n")
	if len(answer.Unresolved) == 0 {
		builder.WriteString("- None recorded by the source answer.\n")
	} else {
		for _, unresolved := range answer.Unresolved {
			fmt.Fprintf(&builder, "- %s\n", unresolved)
		}
	}
	if note := strings.TrimSpace(operatorNote); note != "" {
		builder.WriteString("\n## Operator note\n\n")
		builder.WriteString(note)
		builder.WriteString("\n")
	}
	return builder.String()
}

func renderEvidence(answer qa.Answer) string {
	var builder strings.Builder
	builder.WriteString("# Ask evidence\n\n")
	fmt.Fprintf(&builder, "- Source QA run: `%s`\n", answer.RunID)
	fmt.Fprintf(&builder, "- Question: %s\n", answer.Question)
	fmt.Fprintf(&builder, "- Generated at: `%s`\n\n", answer.GeneratedAt)
	builder.WriteString("## Resolved citations\n\n")
	if len(answer.Citations) == 0 {
		builder.WriteString("- None.\n")
	} else {
		for _, citation := range answer.Citations {
			fmt.Fprintf(&builder, "- `%s` — %s\n", citation.Path, citation.Reason)
		}
	}
	return builder.String()
}
