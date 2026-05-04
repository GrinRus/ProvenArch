package orchestrator

import (
	"strings"

	"github.com/GrinRus/ProvenArch/internal/reports"
)

func artifactIndexFor(artifacts []Artifact) map[string]int {
	index := make(map[string]int, len(artifacts))
	for idx, artifact := range artifacts {
		index[artifact.Kind+"|"+artifact.Path] = idx
	}
	return index
}

func toOrchestratorArtifacts(artifacts []reports.Artifact) []Artifact {
	out := make([]Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, Artifact{
			Path:  artifact.Path,
			Kind:  artifact.Kind,
			Label: artifact.Label,
		})
	}
	return out
}

func toReportArtifacts(artifacts []Artifact) []reports.Artifact {
	out := make([]reports.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, reports.Artifact{
			Path:  artifact.Path,
			Kind:  artifact.Kind,
			Label: artifact.Label,
		})
	}
	return out
}

func (e *pipelineExecution) addArtifacts(artifacts ...Artifact) {
	if e.artifactIndex == nil {
		e.artifactIndex = map[string]int{}
	}
	for _, artifact := range artifacts {
		key := artifact.Kind + "|" + artifact.Path
		if existingIndex, exists := e.artifactIndex[key]; exists {
			e.artifacts[existingIndex] = artifact
			continue
		}
		e.artifactIndex[key] = len(e.artifacts)
		e.artifacts = append(e.artifacts, artifact)
	}
}

func (e *pipelineExecution) removeArtifactsByPath(paths ...string) {
	if len(paths) == 0 || len(e.artifacts) == 0 {
		return
	}
	removeSet := map[string]struct{}{}
	for _, item := range paths {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			removeSet[trimmed] = struct{}{}
		}
	}
	if len(removeSet) == 0 {
		return
	}
	filtered := make([]Artifact, 0, len(e.artifacts))
	for _, artifact := range e.artifacts {
		if _, exists := removeSet[artifact.Path]; exists {
			continue
		}
		filtered = append(filtered, artifact)
	}
	e.artifacts = filtered
	e.artifactIndex = artifactIndexFor(filtered)
}
