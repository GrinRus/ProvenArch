package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/validation"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type Citation struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Response struct {
	Answer     string     `json:"answer"`
	Citations  []Citation `json:"citations"`
	Unresolved []string   `json:"unresolved"`
	Confidence float64    `json:"confidence"`
}

type Answer struct {
	Version     int        `json:"version"`
	RunID       string     `json:"run_id"`
	Question    string     `json:"question"`
	Answer      string     `json:"answer"`
	Citations   []Citation `json:"citations"`
	Unresolved  []string   `json:"unresolved"`
	Confidence  float64    `json:"confidence"`
	Provider    string     `json:"provider"`
	GeneratedAt string     `json:"generated_at"`
}

type ContextPack struct {
	Version     int               `json:"version"`
	RunID       string            `json:"run_id"`
	Question    string            `json:"question"`
	GeneratedAt string            `json:"generated_at"`
	Documents   []ContextDocument `json:"documents"`
}

type ContextDocument struct {
	Path    string `json:"path"`
	Weight  int    `json:"weight"`
	Content string `json:"content"`
}

type Service struct{}

func NewService() Service {
	return Service{}
}

func (service Service) BuildContextPack(ctx context.Context, ws workspace.Root, question string, runID string, generatedAt time.Time) (ContextPack, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return ContextPack{}, fmt.Errorf("question is required")
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	candidates, err := collectCandidates(ctx, ws)
	if err != nil {
		return ContextPack{}, err
	}
	tokens := tokenize(question)
	ranked := rankContextCandidates(candidates, tokens)

	const (
		maxDocuments    = 60
		maxTotalSize    = 2 * 1024 * 1024
		maxDocumentSize = 96 * 1024
	)
	documents := make([]ContextDocument, 0, minInt(len(ranked), maxDocuments))
	totalSize := 0
	for _, candidate := range ranked {
		select {
		case <-ctx.Done():
			return ContextPack{}, ctx.Err()
		default:
		}
		content := candidate.Content
		if len(content) > maxDocumentSize {
			content = content[:maxDocumentSize]
		}
		if totalSize+len(content) > maxTotalSize && len(documents) > 0 {
			continue
		}
		documents = append(documents, ContextDocument{
			Path:    candidate.Path,
			Weight:  candidate.Weight,
			Content: content,
		})
		totalSize += len(content)
		if len(documents) >= maxDocuments {
			break
		}
	}

	return ContextPack{
		Version:     1,
		RunID:       strings.TrimSpace(runID),
		Question:    question,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Documents:   documents,
	}, nil
}

func ParseAnswer(raw []byte) (Answer, error) {
	if err := validation.ValidateRawJSON(validation.QAAnswerSchema, raw); err != nil {
		return Answer{}, fmt.Errorf("qa answer is invalid: %w", err)
	}
	var answer Answer
	if err := json.Unmarshal(raw, &answer); err != nil {
		return Answer{}, fmt.Errorf("decode qa answer: %w", err)
	}
	if err := validateAnswer(answer); err != nil {
		return Answer{}, err
	}
	return answer, nil
}

func ParseContextPack(raw []byte) (ContextPack, error) {
	var pack ContextPack
	if err := json.Unmarshal(raw, &pack); err != nil {
		return ContextPack{}, fmt.Errorf("decode qa context pack: %w", err)
	}
	problems := []string{}
	if pack.Version != 1 {
		problems = append(problems, "version must be 1")
	}
	if strings.TrimSpace(pack.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if strings.TrimSpace(pack.Question) == "" {
		problems = append(problems, "question is required")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(pack.GeneratedAt)); err != nil {
		problems = append(problems, "generated_at must be RFC3339")
	}
	for idx, document := range pack.Documents {
		label := fmt.Sprintf("documents[%d]", idx)
		if _, ok := normalizeWorkspaceRelativeEvidencePath(document.Path); !ok {
			problems = append(problems, label+".path must be workspace-relative")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return ContextPack{}, fmt.Errorf("qa context pack is invalid: %s", strings.Join(problems, "; "))
	}
	return pack, nil
}

func ValidateAnswerFile(path string) (Answer, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Answer{}, err
	}
	return ParseAnswer(raw)
}

func ValidateAnswerAgainstContext(answer Answer, pack ContextPack) error {
	if strings.TrimSpace(answer.RunID) != strings.TrimSpace(pack.RunID) {
		return fmt.Errorf("qa answer run_id %q does not match context pack run_id %q", answer.RunID, pack.RunID)
	}
	if strings.TrimSpace(answer.Question) != strings.TrimSpace(pack.Question) {
		return fmt.Errorf("qa answer question does not match context pack question")
	}
	contextPaths := map[string]struct{}{}
	for _, document := range pack.Documents {
		if normalized, ok := normalizeWorkspaceRelativeEvidencePath(document.Path); ok {
			contextPaths[normalized] = struct{}{}
		}
	}
	for _, citation := range answer.Citations {
		normalized, ok := normalizeWorkspaceRelativeEvidencePath(citation.Path)
		if !ok {
			return fmt.Errorf("qa answer citation path %q is not a workspace-relative path", citation.Path)
		}
		if _, exists := contextPaths[normalized]; !exists {
			return fmt.Errorf("qa answer citation path %q is not present in context pack documents", citation.Path)
		}
	}
	return nil
}

func validateAnswer(answer Answer) error {
	problems := []string{}
	if answer.Version != 1 {
		problems = append(problems, "version must be 1")
	}
	if strings.TrimSpace(answer.RunID) == "" {
		problems = append(problems, "run_id is required")
	}
	if strings.TrimSpace(answer.Question) == "" {
		problems = append(problems, "question is required")
	}
	if strings.TrimSpace(answer.Answer) == "" {
		problems = append(problems, "answer is required")
	}
	if strings.TrimSpace(answer.Provider) == "" {
		problems = append(problems, "provider is required")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(answer.GeneratedAt)); err != nil {
		problems = append(problems, "generated_at must be RFC3339")
	}
	for idx, citation := range answer.Citations {
		label := fmt.Sprintf("citations[%d]", idx)
		if strings.TrimSpace(citation.Path) == "" {
			problems = append(problems, label+".path is required")
		}
		if strings.TrimSpace(citation.Reason) == "" {
			problems = append(problems, label+".reason is required")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("qa answer is invalid: %s", strings.Join(problems, "; "))
	}
	return nil
}

func normalizeWorkspaceRelativeEvidencePath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func (Service) Ask(ctx context.Context, ws workspace.Root, question string) (Response, error) {
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	default:
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return Response{
			Answer:     "Question is empty. Provide a concrete architecture question.",
			Unresolved: []string{"empty question"},
			Confidence: 0,
		}, nil
	}

	tokens := tokenize(question)
	if len(tokens) == 0 {
		return Response{
			Answer:     "Question has insufficient searchable terms. Provide more concrete domain/entity names.",
			Unresolved: []string{"insufficient search terms"},
			Confidence: 0.1,
		}, nil
	}

	candidates, err := collectCandidates(ctx, ws)
	if err != nil {
		return Response{}, err
	}

	scored := []scoredCitation{}
	matchedTokens := map[string]struct{}{}
	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		default:
		}

		if strings.TrimSpace(candidate.Content) == "" {
			continue
		}
		contentLower := strings.ToLower(candidate.Content)
		hits, hitTokens := countTokenHits(contentLower, tokens)
		if hits == 0 {
			continue
		}
		hitTokens = uniqueSortedStrings(hitTokens)
		for _, token := range hitTokens {
			matchedTokens[token] = struct{}{}
		}

		score := hits*100 + candidate.Weight*10 - pathDepthPenalty(candidate.Path)
		scored = append(scored, scoredCitation{
			Citation: Citation{
				Path:   candidate.Path,
				Reason: citationReason(hits, hitTokens),
			},
			Score: score,
			Hits:  hits,
		})
	}

	if len(scored) == 0 {
		return Response{
			Answer: fmt.Sprintf(
				"Not enough indexed workspace evidence to answer confidently yet (scanned %d documents).",
				len(candidates),
			),
			Unresolved: []string{"no supporting artifacts matched the question"},
			Confidence: 0.2,
		}, nil
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			if scored[i].Hits == scored[j].Hits {
				return scored[i].Citation.Path < scored[j].Citation.Path
			}
			return scored[i].Hits > scored[j].Hits
		}
		return scored[i].Score > scored[j].Score
	})

	citations := make([]Citation, 0, len(scored))
	seenPaths := map[string]struct{}{}
	totalHits := 0
	for _, entry := range scored {
		if _, exists := seenPaths[entry.Citation.Path]; exists {
			continue
		}
		seenPaths[entry.Citation.Path] = struct{}{}
		citations = append(citations, entry.Citation)
		totalHits += entry.Hits
		if len(citations) >= 5 {
			break
		}
	}

	missingTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := matchedTokens[token]; ok {
			continue
		}
		missingTokens = append(missingTokens, token)
	}
	sort.Strings(missingTokens)

	unresolved := make([]string, 0, len(missingTokens))
	for _, token := range missingTokens {
		unresolved = append(unresolved, fmt.Sprintf("no supporting evidence for keyword %q", token))
		if len(unresolved) >= 3 {
			break
		}
	}

	confidence := confidenceFromEvidence(totalHits, len(citations), len(unresolved), len(matchedTokens), len(tokens))
	answer := fmt.Sprintf("Workspace evidence matched %d artifact(s): %s", len(citations), citationPathList(citations))
	if len(missingTokens) > 0 {
		answer += fmt.Sprintf(". Missing evidence for %d keyword(s).", len(missingTokens))
	}

	return Response{
		Answer:     answer,
		Citations:  citations,
		Unresolved: unresolved,
		Confidence: confidence,
	}, nil
}

type scoredCitation struct {
	Citation Citation
	Score    int
	Hits     int
}

type indexedDocument struct {
	Path    string
	Content string
	Weight  int
}

func rankContextCandidates(candidates []indexedDocument, tokens []string) []indexedDocument {
	ranked := append([]indexedDocument(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		leftHits, _ := countTokenHits(strings.ToLower(ranked[i].Content), tokens)
		rightHits, _ := countTokenHits(strings.ToLower(ranked[j].Content), tokens)
		leftScore := leftHits*100 + ranked[i].Weight*10 - pathDepthPenalty(ranked[i].Path)
		rightScore := rightHits*100 + ranked[j].Weight*10 - pathDepthPenalty(ranked[j].Path)
		if leftScore == rightScore {
			return ranked[i].Path < ranked[j].Path
		}
		return leftScore > rightScore
	})
	return ranked
}

func tokenize(input string) []string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return nil
	}
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func countTokenHits(content string, tokens []string) (int, []string) {
	hits := 0
	matched := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if strings.Contains(content, token) {
			hits++
			matched = append(matched, token)
		}
	}
	return hits, matched
}

func citationReason(hits int, hitTokens []string) string {
	if len(hitTokens) == 0 {
		return fmt.Sprintf("matched %d keyword(s)", hits)
	}
	preview := hitTokens
	if len(preview) > 3 {
		preview = preview[:3]
	}
	return fmt.Sprintf("matched %d keyword(s): %s", hits, strings.Join(preview, ", "))
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func citationPathList(citations []Citation) string {
	paths := make([]string, 0, len(citations))
	for _, citation := range citations {
		paths = append(paths, filepath.Clean(citation.Path))
	}
	return strings.Join(paths, ", ")
}

func collectCandidates(ctx context.Context, ws workspace.Root) ([]indexedDocument, error) {
	importsRoot := workspaceRelativePath(ws.Manifest.Docs.ImportsPath)
	fixed := []string{
		"charter/overview.md",
		"reports/as-is/overview.md",
		"reports/as-is/service-catalog.md",
		"reports/findings/findings.md",
		"reports/coverage/open-questions.md",
		"reports/coverage/summary.md",
	}
	if importsRoot != "" {
		fixed = append(fixed, path.Join(importsRoot, "index.yaml"))
	}
	walkRoots := []string{
		"charter/cards",
		"model",
		"reports",
		"proposals",
	}
	if importsRoot != "" {
		walkRoots = append(walkRoots, importsRoot)
	}

	paths := map[string]struct{}{}
	for _, path := range fixed {
		paths[path] = struct{}{}
	}

	for _, root := range walkRoots {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		absRoot, err := ws.Resolve(root)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absRoot); err != nil {
			continue
		}
		err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				rel, relErr := filepath.Rel(ws.Path, path)
				if relErr == nil && shouldSkipContextWalkDir(filepath.ToSlash(rel)) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isIndexableExtension(filepath.Ext(entry.Name())) {
				return nil
			}
			rel, err := filepath.Rel(ws.Path, path)
			if err != nil {
				return err
			}
			paths[filepath.ToSlash(rel)] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	out := make([]indexedDocument, 0, len(paths))
	orderedPaths := make([]string, 0, len(paths))
	for path := range paths {
		orderedPaths = append(orderedPaths, path)
	}
	sort.Strings(orderedPaths)

	for _, path := range orderedPaths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		abs, err := ws.Resolve(path)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		if len(content) > 512*1024 {
			continue
		}
		out = append(out, indexedDocument{
			Path:    path,
			Content: string(content),
			Weight:  weightForPath(path, importsRoot),
		})
	}
	return out, nil
}

func shouldSkipContextWalkDir(rel string) bool {
	rel = strings.Trim(filepath.ToSlash(filepath.Clean(rel)), "/")
	switch rel {
	case "reports/taskruns":
		return true
	default:
		return false
	}
}

func workspaceRelativePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = "./docs/imports"
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "." || cleaned == "" {
		return ""
	}
	return cleaned
}

func isIndexableExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".md", ".markdown", ".txt", ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func weightForPath(path string, importsRoot string) int {
	switch {
	case strings.HasPrefix(path, "reports/findings/"):
		return 8
	case strings.HasPrefix(path, "reports/coverage/"):
		return 7
	case strings.HasPrefix(path, "reports/as-is/"):
		return 6
	case strings.HasPrefix(path, "model/"):
		return 6
	case strings.HasPrefix(path, "charter/cards/"):
		return 5
	case strings.HasPrefix(path, "charter/"):
		return 4
	case importsRoot != "" && (path == importsRoot || strings.HasPrefix(path, strings.TrimSuffix(importsRoot, "/")+"/")):
		return 3
	default:
		return 1
	}
}

func pathDepthPenalty(path string) int {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "/"))
}

func confidenceFromEvidence(totalHits int, citationCount int, unresolvedCount int, matchedTokenCount int, totalTokens int) float64 {
	if citationCount == 0 || totalTokens == 0 {
		return 0.2
	}

	matchRatio := float64(matchedTokenCount) / float64(totalTokens)
	citationFactor := float64(minInt(citationCount, 5)) / 5.0
	hitFactor := float64(minInt(totalHits, 12)) / 12.0
	unresolvedPenalty := float64(unresolvedCount) * 0.08

	score := 0.35 + 0.35*matchRatio + 0.2*citationFactor + 0.15*hitFactor - unresolvedPenalty
	return clampConfidence(score)
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
