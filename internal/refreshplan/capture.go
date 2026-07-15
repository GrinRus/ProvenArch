package refreshplan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

type GitRunner interface {
	Run(ctx context.Context, dir string, args ...string) ([]byte, error)
}

type LocalGitRunner struct{}

func (LocalGitRunner) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func CaptureSourceRevisions(ctx context.Context, ws workspace.Root, resolved []workspace.ResolvedRepo, pipeline, runID string, capturedAt time.Time, baseline *SourceRevisions, git GitRunner) SourceRevisions {
	if git == nil {
		git = LocalGitRunner{}
	}
	fingerprint, inputState, inputIssues := analysisInputFingerprint(ws)
	value := SourceRevisions{
		Version: SourceRevisionsVersion, RunID: strings.TrimSpace(runID), Pipeline: strings.TrimSpace(pipeline),
		CapturedAt: capturedAt.UTC().Format(time.RFC3339Nano), AnalysisInputFingerprint: fingerprint,
		AnalysisInputsState: inputState, InputIssues: inputIssues, Repos: []RepoRevision{},
	}
	if baseline != nil {
		id := baseline.RunID
		value.BaselineRunID = &id
	}
	resolvedByName := map[string]workspace.ResolvedRepo{}
	for _, repo := range resolved {
		resolvedByName[repo.Name] = repo
	}
	baselineByName := map[string]RepoRevision{}
	if baseline != nil {
		for _, repo := range baseline.Repos {
			baselineByName[repo.Name] = repo
		}
	}
	for _, configured := range ws.Manifest.Repos {
		repo := RepoRevision{Name: configured.Name, ConfiguredRef: configured.Ref, EffectiveInclude: []string{"**"}, EffectiveExclude: []string{}, WorktreeState: "unavailable", Comparison: "unknown", FallbackReasons: []string{}}
		if configured.Analysis != nil {
			if len(configured.Analysis.Include) > 0 {
				repo.EffectiveInclude = append([]string(nil), configured.Analysis.Include...)
			}
			repo.EffectiveExclude = append([]string(nil), configured.Analysis.Exclude...)
		}
		if configured.GitURL != "" {
			repo.SourceKind, repo.GitURL = "git_url", configured.GitURL
		} else {
			repo.SourceKind, repo.Path = "path", persistedConfiguredPath(configured.Path, configured.Name)
		}
		resolvedRepo, ok := resolvedByName[configured.Name]
		if ok {
			if raw, err := git.Run(ctx, resolvedRepo.Path, "rev-parse", "--verify", "HEAD^{commit}"); err == nil {
				sha := strings.TrimSpace(string(raw))
				repo.CurrentRevision = &sha
			} else {
				repo.FallbackReasons = append(repo.FallbackReasons, "current_revision_unavailable")
			}
			if raw, err := git.Run(ctx, resolvedRepo.Path, "status", "--porcelain=v1", "--untracked-files=normal"); err == nil {
				if strings.TrimSpace(string(raw)) == "" {
					repo.WorktreeState = "clean"
				} else {
					repo.WorktreeState = "dirty"
					repo.FallbackReasons = append(repo.FallbackReasons, "dirty_worktree")
				}
			} else if !contains(repo.FallbackReasons, "current_revision_unavailable") {
				repo.FallbackReasons = append(repo.FallbackReasons, "current_revision_unavailable")
			}
		} else {
			repo.FallbackReasons = append(repo.FallbackReasons, "current_revision_unavailable")
		}
		prior, hasPrior := baselineByName[configured.Name]
		if !hasPrior {
			if baseline == nil && pipeline == "init" {
				repo.Comparison = "initial"
				repo.FallbackReasons = append(repo.FallbackReasons, "initial_run")
			} else {
				repo.FallbackReasons = append(repo.FallbackReasons, "baseline_missing")
			}
		} else {
			repo.BaselineRevision = prior.CurrentRevision
			switch {
			case repo.CurrentRevision == nil:
				repo.Comparison = "unknown"
			case repo.BaselineRevision == nil:
				repo.Comparison = "unknown"
				repo.FallbackReasons = append(repo.FallbackReasons, "baseline_revision_unavailable")
			case *repo.CurrentRevision == *repo.BaselineRevision:
				repo.Comparison = "unchanged"
			case ok:
				if _, err := git.Run(ctx, resolvedRepo.Path, "merge-base", "--is-ancestor", *repo.BaselineRevision, *repo.CurrentRevision); err == nil {
					repo.Comparison = "ahead"
				} else {
					repo.Comparison = "diverged"
					repo.FallbackReasons = append(repo.FallbackReasons, "history_rewritten")
				}
			}
		}
		repo.FallbackReasons = uniqueSorted(repo.FallbackReasons...)
		value.Repos = append(value.Repos, repo)
	}
	normalizeSourceRevisions(&value)
	return value
}

func persistedConfiguredPath(configured, name string) string {
	configured = filepath.ToSlash(filepath.Clean(strings.TrimSpace(configured)))
	if configured == "." || configured == "" {
		return "."
	}
	if !filepath.IsAbs(configured) {
		return configured
	}
	sum := sha256.Sum256([]byte(configured))
	return "external/" + strings.TrimSpace(name) + "-" + hex.EncodeToString(sum[:8])
}

type fingerprintFile struct{ Path, Hash string }
type fingerprintRepo struct {
	Name, Path, GitURL, Ref string
	Include, Exclude        []string
}
type fingerprintManifest struct {
	Repos       []fingerprintRepo
	ImportsPath string
	Files       []fingerprintFile
}

func analysisInputFingerprint(ws workspace.Root) (string, string, []string) {
	manifest := fingerprintManifest{ImportsPath: filepath.ToSlash(filepath.Clean(ws.Manifest.Docs.ImportsPath)), Repos: []fingerprintRepo{}, Files: []fingerprintFile{}}
	for _, repo := range ws.Manifest.Repos {
		entry := fingerprintRepo{Name: repo.Name, Path: repo.Path, GitURL: repo.GitURL, Ref: repo.Ref, Include: []string{}, Exclude: []string{}}
		if repo.Analysis != nil {
			entry.Include = append(entry.Include, repo.Analysis.Include...)
			entry.Exclude = append(entry.Exclude, repo.Analysis.Exclude...)
		}
		sort.Strings(entry.Include)
		sort.Strings(entry.Exclude)
		manifest.Repos = append(manifest.Repos, entry)
	}
	sort.Slice(manifest.Repos, func(i, j int) bool { return manifest.Repos[i].Name < manifest.Repos[j].Name })
	roots := []string{strings.TrimPrefix(manifest.ImportsPath, "./"), "charter", "skills"}
	state := "complete"
	issues := []string{}
	for _, root := range roots {
		abs, err := ws.Resolve(root)
		if err != nil {
			state = "unavailable"
			issues = append(issues, "analysis_inputs_unreadable")
			manifest.Files = append(manifest.Files, fingerprintFile{Path: root, Hash: "unreadable"})
			continue
		}
		err = filepath.WalkDir(abs, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(ws.Path, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(raw)
			manifest.Files = append(manifest.Files, fingerprintFile{Path: filepath.ToSlash(rel), Hash: hex.EncodeToString(sum[:])})
			return nil
		})
		if os.IsNotExist(err) {
			manifest.Files = append(manifest.Files, fingerprintFile{Path: root, Hash: "missing"})
			continue
		}
		if err != nil {
			state = "unavailable"
			issues = append(issues, "analysis_inputs_unreadable")
			manifest.Files = append(manifest.Files, fingerprintFile{Path: root, Hash: "unreadable"})
		}
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	raw, _ := json.Marshal(manifest)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), state, uniqueSorted(issues...)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
