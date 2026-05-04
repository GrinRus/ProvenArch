package workspace

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type docsImportIndexEntry struct {
	ID              string `yaml:"id"`
	Path            string `yaml:"path"`
	Source          string `yaml:"source,omitempty"`
	Checksum        string `yaml:"checksum,omitempty"`
	ImportedAt      string `yaml:"imported_at,omitempty"`
	SourceUpdatedAt string `yaml:"source_updated_at,omitempty"`
	Status          string `yaml:"status,omitempty"`
}

func (r Root) validateDocsImportsIndex(importsRootAbs string) []Diagnostic {
	indexPath := filepath.Join(importsRootAbs, "index.yaml")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Diagnostic{{
			Level:      DiagnosticWarning,
			Code:       "workspace.docs.imports_index.unreadable",
			Message:    fmt.Sprintf("cannot read docs imports index %q: %v", indexPath, err),
			Path:       indexPath,
			Suggestion: "Fix file permissions or remove the unreadable imports index",
		}}
	}

	var entries []docsImportIndexEntry
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&entries); err != nil {
		return []Diagnostic{docsImportsIndexWarning(
			"workspace.docs.imports_index.malformed",
			indexPath,
			fmt.Sprintf("docs imports index is malformed: %v", err),
			"Use a YAML list of entries with required id and path fields",
		)}
	}
	if err := decoder.Decode(&struct{}{}); !isEOF(err) {
		return []Diagnostic{docsImportsIndexWarning(
			"workspace.docs.imports_index.malformed",
			indexPath,
			"docs imports index must contain a single YAML document",
			"Keep all imports metadata entries in one YAML list",
		)}
	}

	warnings := []Diagnostic{}
	seenIDs := map[string]struct{}{}
	importsRootClean := filepath.Clean(importsRootAbs)
	for idx, entry := range entries {
		entryPath := strings.TrimSpace(entry.Path)
		entryID := strings.TrimSpace(entry.ID)
		display := fmt.Sprintf("%s[%d]", indexPath, idx)

		if entryID == "" {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.entry_id_required",
				indexPath,
				fmt.Sprintf("docs imports index entry %d must include id", idx),
				"Add a stable id for each imports metadata entry",
			))
		} else if _, exists := seenIDs[entryID]; exists {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.duplicate_id",
				indexPath,
				fmt.Sprintf("docs imports index id %q is duplicated", entryID),
				"Use unique id values for imports metadata entries",
			))
		} else {
			seenIDs[entryID] = struct{}{}
		}

		if entryPath == "" {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.entry_path_required",
				indexPath,
				fmt.Sprintf("docs imports index entry %s must include path", display),
				"Add a workspace-relative path under docs.imports_path",
			))
			continue
		}
		if filepath.IsAbs(entryPath) {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.path_absolute",
				indexPath,
				fmt.Sprintf("docs imports index path %q must be workspace-relative", entryPath),
				"Use a workspace-relative path under docs.imports_path",
			))
			continue
		}
		resolvedPath, err := r.Resolve(entryPath)
		if err != nil {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.path_invalid",
				indexPath,
				fmt.Sprintf("docs imports index path %q is invalid: %v", entryPath, err),
				"Use a workspace-relative path under docs.imports_path",
			))
			continue
		}
		if !pathWithinRoot(resolvedPath, importsRootClean) {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.path_outside_imports",
				indexPath,
				fmt.Sprintf("docs imports index path %q must resolve under docs.imports_path", entryPath),
				"Move the referenced document under docs.imports_path or update the path",
			))
			continue
		}
		if info, err := os.Stat(resolvedPath); err != nil {
			if os.IsNotExist(err) {
				warnings = append(warnings, docsImportsIndexWarning(
					"workspace.docs.imports_index.path_missing",
					indexPath,
					fmt.Sprintf("docs imports index path %q does not exist", entryPath),
					"Add the referenced import file or remove the stale metadata entry",
				))
			} else {
				warnings = append(warnings, docsImportsIndexWarning(
					"workspace.docs.imports_index.path_unreadable",
					indexPath,
					fmt.Sprintf("cannot access docs imports index path %q: %v", entryPath, err),
					"Fix file permissions for the referenced import file",
				))
			}
		} else if info.IsDir() {
			warnings = append(warnings, docsImportsIndexWarning(
				"workspace.docs.imports_index.path_not_file",
				indexPath,
				fmt.Sprintf("docs imports index path %q must reference a file, not a directory", entryPath),
				"Point the metadata entry to an imported document file",
			))
		}
	}
	return warnings
}

func docsImportsIndexWarning(code string, path string, message string, suggestion string) Diagnostic {
	return Diagnostic{
		Level:      DiagnosticWarning,
		Code:       code,
		Message:    message,
		Path:       path,
		Suggestion: suggestion,
	}
}

func pathWithinRoot(path string, root string) bool {
	rel, err := filepath.Rel(root, filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func isEOF(err error) bool {
	return err == io.EOF
}
