package orchestrator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/slugutil"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const step0WizardContractPath = "charter/wizard/step0-contract.json"

type step0WizardContract struct {
	Version       int      `json:"version"`
	ProjectName   string   `json:"project_name"`
	Scope         string   `json:"scope"`
	NFRPriorities []string `json:"nfr_priorities"`
	Rules         []string `json:"rules"`
}

func loadStep0WizardContract(ws workspace.Root) (step0WizardContract, bool, error) {
	content, err := ws.ReadFile(step0WizardContractPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return step0WizardContract{}, false, nil
		}
		return step0WizardContract{}, false, fmt.Errorf("read %s: %w", step0WizardContractPath, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()

	var contract step0WizardContract
	if err := decoder.Decode(&contract); err != nil {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: %w", step0WizardContractPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: unexpected trailing JSON payload", step0WizardContractPath)
	} else if !errors.Is(err, io.EOF) {
		return step0WizardContract{}, true, fmt.Errorf("parse %s: %w", step0WizardContractPath, err)
	}

	contract.ProjectName = strings.TrimSpace(contract.ProjectName)
	contract.Scope = strings.TrimSpace(contract.Scope)
	contract.NFRPriorities = normalizeOrderedUniqueStrings(contract.NFRPriorities)
	contract.Rules = normalizeOrderedUniqueStrings(contract.Rules)

	validationProblems := []string{}
	if contract.Version != 1 {
		validationProblems = append(validationProblems, "version must be 1")
	}
	if contract.ProjectName == "" {
		validationProblems = append(validationProblems, "project_name is required")
	}
	if contract.Scope == "" {
		validationProblems = append(validationProblems, "scope is required")
	}
	if len(validationProblems) > 0 {
		sort.Strings(validationProblems)
		return step0WizardContract{}, true, fmt.Errorf("invalid %s: %s", step0WizardContractPath, strings.Join(validationProblems, "; "))
	}

	return contract, true, nil
}

func (e *pipelineExecution) publishValidatedConstitutionDrafts(execution runtimeTaskExecution) error {
	draft, _, err := validateRequiredRuntimeDraftArtifacts(execution.Task)
	if err != nil {
		return err
	}
	e.step0DraftManifest = &draft
	e.step0DraftRoot = execution.Task.DraftFinalRoot
	e.addArtifacts(Artifact{
		Path:  path.Join(execution.Task.ArtifactRoot, constitutionDraftManifestFile),
		Kind:  "taskrun",
		Label: "Constitution Draft Manifest",
	})
	publishedDrafts, err := applyRuntimeDraftOutputs(
		e.workspace,
		execution.Task.DraftFinalRoot,
		draft,
		"",
		func(canonicalPath string) bool {
			switch canonicalPath {
			case "charter/overview.md", "skills/subagents.yaml":
				return true
			default:
				return false
			}
		},
	)
	if err != nil {
		return fmt.Errorf("publish constitution runtime drafts: %w", err)
	}
	e.addArtifacts(publishedDrafts...)
	return nil
}

func (e *pipelineExecution) materializeConstitutionSupportSurface(stepID string) error {
	e.logInfo(stepID, "", "materializing constitution support artifacts", nil)
	step0Contract, hasStep0Contract, err := loadStep0WizardContract(e.workspace)
	if err != nil {
		e.addWarning(fmt.Sprintf("step0_wizard_contract_invalid: %v; fallback baseline constitution support artifacts are used", err))
	}
	if !hasStep0Contract {
		e.addWarning("step0_wizard_contract_missing: charter/wizard/step0-contract.json not found; fallback baseline constitution support artifacts are used")
	}
	if err := e.writeConstitutionSupportArtifacts(hasStep0Contract && err == nil, step0Contract); err != nil {
		return err
	}
	if err := writeBaselineSupportBundle(e.workspace); err != nil {
		return err
	}
	return nil
}

func (e *pipelineExecution) writeConstitutionSupportArtifacts(useWizardContract bool, contract step0WizardContract) error {
	projectName := ""
	scope := ""
	nfrPriorities := []string{}
	rules := []string{}
	if useWizardContract {
		projectName = contract.ProjectName
		scope = contract.Scope
		nfrPriorities = append([]string(nil), contract.NFRPriorities...)
		rules = append([]string(nil), contract.Rules...)
	}

	glossary := "terms: []\n"
	if useWizardContract {
		scopeTerms := splitAndNormalizeList(scope)
		glossary = renderStringListYAML("terms", scopeTerms)
	}
	nfrContent := renderStringListYAML("nfr", nfrPriorities)
	rulesContent := renderStringListYAML("rules", rules)

	if err := e.workspace.WriteFile("charter/glossary.yaml", []byte(glossary)); err != nil {
		return err
	}
	if err := e.workspace.WriteFile("charter/nfr.yaml", []byte(nfrContent)); err != nil {
		return err
	}
	if err := e.workspace.WriteFile("charter/rules.yaml", []byte(rulesContent)); err != nil {
		return err
	}

	for _, repo := range e.workspace.Manifest.Repos {
		slug := slugutil.Slugify(repo.Name)
		domainPath := fmt.Sprintf("charter/cards/domains/%s.md", slug)
		domainBody := strings.TrimSpace(fmt.Sprintf("# Domain: %s\n\n- id: `%s`\n- repo_scope: `%s`\n", repo.Name, slug, repo.Name))
		if useWizardContract {
			domainBody += fmt.Sprintf("\n- charter_project: `%s`\n- charter_scope: `%s`\n", projectName, scope)
		}
		domainBody += "\n"
		if err := e.workspace.WriteFile(domainPath, []byte(domainBody)); err != nil {
			return err
		}
	}

	teamBody := "# Team: Platform\n\n- id: `team.platform`\n"
	if useWizardContract {
		teamBody = strings.TrimSpace(teamBody+fmt.Sprintf("- charter_project: `%s`\n", projectName)) + "\n"
	}
	if err := e.workspace.WriteFile("charter/cards/teams/platform.md", []byte(teamBody)); err != nil {
		return err
	}

	return nil
}

func renderStringListYAML(key string, values []string) string {
	values = normalizeOrderedUniqueStrings(values)
	if len(values) == 0 {
		return fmt.Sprintf("%s: []\n", strings.TrimSpace(key))
	}

	builder := strings.Builder{}
	builder.WriteString(strings.TrimSpace(key))
	builder.WriteString(":\n")
	for _, value := range values {
		builder.WriteString("  - ")
		builder.WriteString(strconv.Quote(value))
		builder.WriteString("\n")
	}
	return builder.String()
}

func splitAndNormalizeList(raw string) []string {
	replacer := strings.NewReplacer(",", "\n", ";", "\n", "\t", "\n")
	values := strings.Split(replacer.Replace(raw), "\n")
	return normalizeOrderedUniqueStrings(values)
}

func normalizeOrderedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
