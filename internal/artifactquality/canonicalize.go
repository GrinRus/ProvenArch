package artifactquality

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func ValidateCollectManifest(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
	if writeRoot == "" {
		return fmt.Errorf("collect write_root is empty")
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	return ValidateCollectManifestBytes(raw)
}

func ValidateCollectManifestBytes(raw []byte) error {
	if violations := detectLegacyCollectManifestViolations(raw); len(violations) > 0 {
		return fmt.Errorf(
			"legacy collect manifest fields are forbidden: %s",
			strings.Join(violations, "; "),
		)
	}
	_, err := contracts.ParseShardPackManifest(raw)
	return err
}

func detectLegacyCollectManifestViolations(raw []byte) []string {
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	root, ok := payload.(map[string]any)
	if !ok {
		return nil
	}

	violations := make([]string, 0, 8)
	appendViolation := func(path string, detail string) {
		path = strings.TrimSpace(path)
		detail = strings.TrimSpace(detail)
		if path == "" || detail == "" {
			return
		}
		violation := fmt.Sprintf("%s (%s)", path, detail)
		for _, existing := range violations {
			if existing == violation {
				return
			}
		}
		violations = append(violations, violation)
	}

	if _, exists := root["step_contract"]; exists {
		appendViolation("step_contract", "top-level step_contract is forbidden")
	}
	if _, exists := root["compatibility"]; exists {
		appendViolation("compatibility", "legacy compatibility block is forbidden")
	}

	semantic, _ := root["semantic"].(map[string]any)
	if coverage, ok := semantic["coverage"].(map[string]any); ok {
		if _, exists := coverage["covered_topics"]; exists {
			appendViolation("semantic.coverage.covered_topics", "use semantic.coverage.observed")
		}
	}
	if questions, ok := semantic["questions"].([]any); ok {
		for index, question := range questions {
			questionMap, ok := question.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := questionMap["question"]; exists {
				appendViolation(fmt.Sprintf("semantic.questions[%d].question", index), "use semantic.questions[*].text")
			}
			if confidence, exists := questionMap["confidence"]; exists {
				appendStringConfidenceViolation(appendViolation, fmt.Sprintf("semantic.questions[%d].confidence", index), confidence)
			}
		}
	}
	if entities, ok := semantic["entities"].([]any); ok {
		for index, entity := range entities {
			entityMap, ok := entity.(map[string]any)
			if !ok {
				continue
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.entities[%d].provenance", index), entityMap["provenance"])
		}
	}
	if edges, ok := semantic["edges"].([]any); ok {
		for index, edge := range edges {
			edgeMap, ok := edge.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := edgeMap["relation"]; exists {
				appendViolation(fmt.Sprintf("semantic.edges[%d].relation", index), "use semantic.edges[*].type")
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.edges[%d].provenance", index), edgeMap["provenance"])
		}
	}
	if findings, ok := semantic["findings"].([]any); ok {
		for index, finding := range findings {
			findingMap, ok := finding.(map[string]any)
			if !ok {
				continue
			}
			if _, exists := findingMap["evidence_citation_ids"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].evidence_citation_ids", index), "use semantic.findings[*].provenance.evidence")
			}
			if _, exists := findingMap["inference"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].inference", index), "legacy inference compatibility fields are forbidden")
			}
			if _, exists := findingMap["summary"]; exists {
				appendViolation(fmt.Sprintf("semantic.findings[%d].summary", index), "legacy summary compatibility fields are forbidden")
			}
			if confidence, exists := findingMap["confidence"]; exists {
				appendStringConfidenceViolation(appendViolation, fmt.Sprintf("semantic.findings[%d].confidence", index), confidence)
			}
			detectLegacyProvenanceShape(appendViolation, fmt.Sprintf("semantic.findings[%d].provenance", index), findingMap["provenance"])
		}
	}

	if len(violations) > 8 {
		return append(violations[:8], "additional legacy collect violations omitted")
	}
	return violations
}

func appendStringConfidenceViolation(appendViolation func(string, string), path string, value any) {
	if _, ok := value.(string); ok {
		appendViolation(path, "confidence must be numeric")
	}
}

func detectLegacyProvenanceShape(appendViolation func(string, string), path string, value any) {
	switch typed := value.(type) {
	case []any:
		appendViolation(path, "provenance must be an object, not an array")
	case map[string]any:
		if confidence, exists := typed["confidence"]; exists {
			appendStringConfidenceViolation(appendViolation, path+".confidence", confidence)
		}
		if evidence, exists := typed["evidence"].([]any); exists {
			for index, item := range evidence {
				detectSemanticEvidenceViolations(appendViolation, fmt.Sprintf("%s.evidence[%d]", path, index), item)
			}
		}
	}
}

func detectSemanticEvidenceViolations(appendViolation func(string, string), path string, value any) {
	item, ok := value.(map[string]any)
	if !ok {
		return
	}
	repo := strings.TrimSpace(stringValue(item["repo"]))
	evidencePath := strings.TrimSpace(stringValue(item["path"]))
	if repo == "" || evidencePath == "" {
		appendViolation(path, "semantic provenance evidence requires non-empty repo/path; citation-only evidence objects are forbidden")
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
