package workspace

import (
	"path/filepath"
	"strings"
	"unicode"
)

type repoRoleResolution struct {
	DeclaredRole  string
	EffectiveRole string
	Source        string
}

var frontendRepoRoleTokens = map[string]struct{}{
	"android":  {},
	"browser":  {},
	"client":   {},
	"clients":  {},
	"frontend": {},
	"ios":      {},
	"mobile":   {},
	"ui":       {},
	"web":      {},
	"website":  {},
}

var backendRepoRoleTokens = map[string]struct{}{
	"api":      {},
	"backend":  {},
	"daemon":   {},
	"gateway":  {},
	"server":   {},
	"service":  {},
	"services": {},
	"worker":   {},
}

func ResolveRepoRole(repo RepoSource) repoRoleResolution {
	declaredRole := ""
	if repo.Analysis != nil {
		declaredRole = strings.TrimSpace(strings.ToLower(repo.Analysis.Role))
	}
	if normalized, ok := NormalizeRepoRole(declaredRole); ok {
		return repoRoleResolution{
			DeclaredRole:  declaredRole,
			EffectiveRole: normalized,
			Source:        "declared",
		}
	}

	effectiveRole := inferRepoRoleFromSource(repo)
	if effectiveRole != RepoRoleUnknown {
		return repoRoleResolution{
			DeclaredRole:  declaredRole,
			EffectiveRole: effectiveRole,
			Source:        "inferred",
		}
	}

	return repoRoleResolution{
		DeclaredRole:  declaredRole,
		EffectiveRole: RepoRoleUnknown,
		Source:        "unknown",
	}
}

func inferRepoRoleFromSource(repo RepoSource) string {
	tokens := repoRoleTokens(repo)
	if len(tokens) == 0 {
		return RepoRoleUnknown
	}

	frontendHit := hasRepoRoleToken(tokens, frontendRepoRoleTokens)
	backendHit := hasRepoRoleToken(tokens, backendRepoRoleTokens)

	switch {
	case frontendHit && !backendHit:
		return RepoRoleFrontend
	case backendHit && !frontendHit:
		return RepoRoleBackend
	case frontendHit && backendHit:
		return RepoRoleMixed
	default:
		return RepoRoleUnknown
	}
}

func repoRoleTokens(repo RepoSource) map[string]struct{} {
	candidates := []string{
		strings.TrimSpace(repo.Name),
		strings.TrimSpace(repo.Path),
		strings.TrimSpace(repo.GitURL),
	}

	tokens := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		for _, part := range repoRoleCandidateParts(candidate) {
			for _, token := range strings.FieldsFunc(strings.ToLower(part), func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r)
			}) {
				token = strings.TrimSpace(token)
				if token == "" {
					continue
				}
				tokens[token] = struct{}{}
			}
		}
	}
	return tokens
}

func repoRoleCandidateParts(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	base := filepath.Base(value)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	parts := []string{value}
	if base != "" && base != value {
		parts = append(parts, base)
	}
	return parts
}

func hasRepoRoleToken(tokens map[string]struct{}, expected map[string]struct{}) bool {
	for token := range tokens {
		if _, ok := expected[token]; ok {
			return true
		}
	}
	return false
}
