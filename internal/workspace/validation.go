package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
)

type DiagnosticLevel string

const (
	DiagnosticError   DiagnosticLevel = "error"
	DiagnosticWarning DiagnosticLevel = "warning"
)

type Diagnostic struct {
	Level      DiagnosticLevel `json:"level"`
	Code       string          `json:"code"`
	Message    string          `json:"message"`
	Repo       string          `json:"repo,omitempty"`
	Path       string          `json:"path,omitempty"`
	Suggestion string          `json:"suggestion,omitempty"`
}

type ValidationReport struct {
	Workspace     string         `json:"workspace"`
	OK            bool           `json:"ok"`
	Errors        []Diagnostic   `json:"errors,omitempty"`
	Warnings      []Diagnostic   `json:"warnings,omitempty"`
	ResolvedRepos []ResolvedRepo `json:"resolved_repos,omitempty"`
}

type ValidateOptions struct {
	ResolveRepos bool
	FetchGit     bool
	VerifyRefs   bool
}

func (r Root) Validate(ctx context.Context, options ValidateOptions) ValidationReport {
	report := ValidationReport{Workspace: r.Path}

	importsPath, err := r.Resolve(r.Manifest.Docs.ImportsPath)
	if err != nil {
		report.Errors = append(report.Errors, Diagnostic{
			Level:      DiagnosticError,
			Code:       "workspace.docs.imports_path.invalid",
			Message:    fmt.Sprintf("docs.imports_path %q is invalid: %v", r.Manifest.Docs.ImportsPath, err),
			Suggestion: "Use a workspace-relative imports path, e.g. ./docs/imports",
		})
	} else {
		if info, statErr := os.Stat(importsPath); statErr != nil {
			if os.IsNotExist(statErr) {
				report.Warnings = append(report.Warnings, Diagnostic{
					Level:      DiagnosticWarning,
					Code:       "workspace.docs.imports_path.missing",
					Message:    fmt.Sprintf("docs imports directory %q does not exist yet", r.Manifest.Docs.ImportsPath),
					Path:       importsPath,
					Suggestion: "Create the directory or run init pipeline to bootstrap layout",
				})
			} else {
				report.Errors = append(report.Errors, Diagnostic{
					Level:      DiagnosticError,
					Code:       "workspace.docs.imports_path.unreadable",
					Message:    fmt.Sprintf("cannot access docs imports directory: %v", statErr),
					Path:       importsPath,
					Suggestion: "Fix filesystem permissions for the workspace",
				})
			}
		} else if !info.IsDir() {
			report.Errors = append(report.Errors, Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.docs.imports_path.not_dir",
				Message:    fmt.Sprintf("docs.imports_path %q must point to a directory", r.Manifest.Docs.ImportsPath),
				Path:       importsPath,
				Suggestion: "Update workspace.yaml docs.imports_path to a directory",
			})
		}
	}

	if options.ResolveRepos {
		resolved, diagnostics := r.ResolveRepoSources(ctx, ResolveOptions{
			FetchGit:   options.FetchGit,
			VerifyRefs: options.VerifyRefs,
		})
		report.ResolvedRepos = resolved
		for _, diagnostic := range diagnostics {
			if diagnostic.Level == DiagnosticError {
				report.Errors = append(report.Errors, diagnostic)
			} else {
				report.Warnings = append(report.Warnings, diagnostic)
			}
		}
	}

	for _, diagnostic := range r.validateSubagentsBundle() {
		if diagnostic.Level == DiagnosticError {
			report.Errors = append(report.Errors, diagnostic)
		} else {
			report.Warnings = append(report.Warnings, diagnostic)
		}
	}
	for _, diagnostic := range r.validateBaselineBundleManifest() {
		if diagnostic.Level == DiagnosticError {
			report.Errors = append(report.Errors, diagnostic)
		} else {
			report.Warnings = append(report.Warnings, diagnostic)
		}
	}
	for _, diagnostic := range r.validateLayoutReadiness() {
		if diagnostic.Level == DiagnosticError {
			report.Errors = append(report.Errors, diagnostic)
		} else {
			report.Warnings = append(report.Warnings, diagnostic)
		}
	}

	report.OK = len(report.Errors) == 0
	return report
}

func (r ValidationReport) Error() string {
	if r.OK {
		return ""
	}
	if len(r.Errors) == 0 {
		return "workspace validation failed"
	}
	return r.Errors[0].Message
}

func (r Root) validateLayoutReadiness() []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(requiredLayoutDirs))
	for _, rel := range requiredLayoutDirs {
		abs, err := r.Resolve(rel)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.layout.path.invalid",
				Message:    fmt.Sprintf("layout path %q is invalid: %v", rel, err),
				Suggestion: "Keep workspace root and layout paths inside the workspace boundary",
			})
			continue
		}
		info, statErr := os.Stat(abs)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				diagnostics = append(diagnostics, Diagnostic{
					Level:      DiagnosticWarning,
					Code:       "workspace.layout.dir.missing",
					Path:       abs,
					Message:    fmt.Sprintf("layout directory %q is missing and will be created on run", rel),
					Suggestion: "Run init/refresh pipeline to materialize workspace layout",
				})
				continue
			}
			diagnostics = append(diagnostics, Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.layout.dir.unreadable",
				Path:       abs,
				Message:    fmt.Sprintf("cannot access layout directory %q: %v", rel, statErr),
				Suggestion: "Fix filesystem permissions for the workspace",
			})
			continue
		}
		if !info.IsDir() {
			diagnostics = append(diagnostics, Diagnostic{
				Level:      DiagnosticError,
				Code:       "workspace.layout.dir.not_dir",
				Path:       abs,
				Message:    fmt.Sprintf("layout path %q must be a directory", rel),
				Suggestion: "Remove conflicting file and create directory for workspace layout",
			})
		}
	}
	return diagnostics
}
