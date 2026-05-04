package qa

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

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

type Service struct{}

func NewService() Service {
	return Service{}
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
