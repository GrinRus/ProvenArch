package orchestrator

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/refreshplan"
)

const maxRefreshIntentBytes = 64 * 1024

func buildRefreshIntentContext(ctx context.Context, impact refreshplan.ImpactPlan, repoPaths map[string]string) string {
	sections := []string{"REFRESH AFFECTED-SCOPE CONTEXT", "Current source files and observed evidence are authoritative. Commit subjects are secondary intent hints and must never override source evidence."}
	deltas := append([]refreshplan.RepoDelta(nil), impact.RepoDeltas...)
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].Repo < deltas[j].Repo })
	for _, delta := range deltas {
		lines := []string{fmt.Sprintf("repo=%s", delta.Repo)}
		for _, change := range delta.Changes {
			if change.InScope {
				lines = append(lines, fmt.Sprintf("changed=%s:%s", change.Status, change.Path))
			}
		}
		if repoPath := strings.TrimSpace(repoPaths[delta.Repo]); repoPath != "" && delta.BaselineRevision != nil && delta.CurrentRevision != nil {
			output, err := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "--max-count=20", "--format=%H%x09%s", *delta.BaselineRevision+".."+*delta.CurrentRevision).Output()
			if err == nil {
				for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
					parts := strings.SplitN(line, "\t", 2)
					if len(parts) == 2 {
						subject := []rune(strings.TrimSpace(parts[1]))
						if len(subject) > 200 {
							subject = subject[:200]
						}
						lines = append(lines, fmt.Sprintf("commit=%s %s", parts[0], string(subject)))
					}
				}
			}
		}
		for _, line := range lines {
			if len(strings.Join(append(sections, line), "\n")) > maxRefreshIntentBytes {
				return strings.Join(sections, "\n")
			}
			sections = append(sections, line)
		}
	}
	return strings.Join(sections, "\n")
}
