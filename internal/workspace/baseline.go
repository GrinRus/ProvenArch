package workspace

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
)

var baselineSkillIDs = []string{
	"service-inventory",
	"interface-extraction",
	"integration-mapping",
	"datastore-mapping",
	"cicd-mapping",
	"ownership-coverage",
	"findings",
	"proposals",
	"qa",
}

var baselinePromptPacks = map[string]string{
	"constitution": renderStructuredPrompt("Constitution Prompt Pack", structuredPromptSpec{
		Goal: "Materialize deterministic charter baseline artifacts from wizard contract and canonical templates.",
		Inputs: []string{
			"charter/wizard/step0-contract.json when present and valid",
			"charter templates and existing charter/cards files",
			"workspace.yaml repo list for initial canonical cards",
		},
		RequiredOutputShape: []string{
			"Stable markdown/yaml/json edits only inside charter/* and skills/* baseline surface",
			"No free-form side files, no hidden metadata, no extra sections outside template conventions",
			"Preserve deterministic ordering for list fields and bullet sections",
		},
		EvidencePolicy: []string{
			"Treat wizard contract + workspace manifest as primary evidence",
			"If required values are missing or invalid, keep fallback-safe baseline wording and add explicit unresolved markers",
			"Never assert organizational facts that are absent from workspace evidence",
		},
		ForbiddenBehavior: []string{
			"Do not invent teams, domains, owners, or system boundaries",
			"Do not overwrite user-modified files when create-if-missing policy applies",
			"Do not emit markdown code fences around persisted artifact content",
		},
		FallbackWhenUnknown: []string{
			"Use baseline constitution templates with generic but deterministic placeholders",
			"Propagate unresolved details into coverage/questions instead of fabricated values",
			"Keep output reviewable and minimal when signal is insufficient",
		},
	}),
	"collect-context": renderStructuredPrompt("Collect Context Prompt Pack", structuredPromptSpec{
		Goal: "Extract evidence-backed architecture context for selected repo scopes without fabricating entities or integrations.",
		Inputs: []string{
			"workspace-selected repo scopes and path scopes",
			"source code, CI/CD manifests, deployment configs, docs/imports",
			"charter cards, glossary, and runtime step policy",
		},
		RequiredOutputShape: []string{
			"TaskResult JSON with canonical meta + summary + changeset",
			"Entity and edge operations with explicit provenance/evidence arrays",
			"questions[] and coverage{} blocks for unknowns and missing evidence",
		},
		EvidencePolicy: []string{
			"Observation requires concrete evidence path/repo references",
			"Inference must be explicitly marked and confidence-scored",
			"Prefer precise file-level evidence over broad repository claims",
		},
		ForbiddenBehavior: []string{
			"Do not output plain prose outside TaskResult JSON envelope",
			"Do not guess runtime metrics, ownership, or external contracts without evidence",
			"Do not introduce non-deterministic timestamps or unstable identifiers outside allowed meta fields",
		},
		FallbackWhenUnknown: []string{
			"Emit canonical coverage.missing values and actionable questions",
			"Keep partial but valid TaskResult even under sparse repository signal",
			"Flag critical gaps explicitly instead of silent omission",
		},
	}),
	"findings": renderStructuredPrompt("Findings Prompt Pack", structuredPromptSpec{
		Goal: "Produce deterministic, evidence-linked architecture findings from model state and charter rules.",
		Inputs: []string{
			"model/entities and model/edges materialized by previous steps",
			"charter/rules.yaml and coverage/open-questions context",
			"runtime task scopes and prior domain outputs",
		},
		RequiredOutputShape: []string{
			"TaskResult JSON with add_finding operations in changeset",
			"Each finding includes stable id/title/severity/description and provenance",
			"Optional coverage/questions updates when evidence is incomplete",
		},
		EvidencePolicy: []string{
			"Findings must point to concrete evidence references or explicit inference rationale",
			"Severity should reflect impact and confidence, not stylistic preferences",
			"Cross-repo claims require cross-repo evidence",
		},
		ForbiddenBehavior: []string{
			"Do not report speculative anti-patterns without source evidence",
			"Do not duplicate semantically identical findings under different ids",
			"Do not include remediation plans in finding description unless mandated by output shape",
		},
		FallbackWhenUnknown: []string{
			"When proof is weak, emit high-priority question rather than hard finding",
			"If owner linkage is missing, surface owner-gap finding with explicit uncertainty",
			"Prefer fewer high-signal findings over noisy exhaustive lists",
		},
	}),
	"proposals": renderStructuredPrompt("Proposals Prompt Pack", structuredPromptSpec{
		Goal: "Compile deterministic implementation proposals mapped to findings and workspace constraints.",
		Inputs: []string{
			"reports/findings/findings.md and related model context",
			"charter constraints, NFR priorities, and migration boundaries",
			"existing proposal templates and ADR/RFC stubs",
		},
		RequiredOutputShape: []string{
			"Structured proposal artifacts with explicit scope, rollout, and risk notes",
			"Deterministic ordering of finding references and migration checklist items",
			"No contract-breaking fields outside current proposal artifact conventions",
		},
		EvidencePolicy: []string{
			"Every proposal item references one or more concrete findings or coverage gaps",
			"State assumptions explicitly when evidence cannot prove implementation details",
			"Keep changes aligned with observed architecture constraints",
		},
		ForbiddenBehavior: []string{
			"Do not propose features unrelated to current findings scope",
			"Do not promise runtime/security/compliance guarantees outside MVP boundary",
			"Do not mutate schema contracts from proposal generation step",
		},
		FallbackWhenUnknown: []string{
			"When effort or ownership is unknown, add deterministic TODO markers",
			"Prefer phased rollout skeleton over speculative detailed migrations",
			"Preserve reviewability with concise, numbered action plans",
		},
	}),
	"qa": renderStructuredPrompt("QA Prompt Pack", structuredPromptSpec{
		Goal: "Answer read-only architecture questions using workspace artifacts with explicit citations and uncertainty handling.",
		Inputs: []string{
			"charter/cards, model, reports, docs/imports, and run artifacts",
			"user question and optional scope hints",
			"current evidence policy and confidence model",
		},
		RequiredOutputShape: []string{
			"Short answer followed by citation set (repo/path oriented)",
			"Unresolved section when evidence is missing or contradictory",
			"Confidence statement calibrated to evidence quality",
		},
		EvidencePolicy: []string{
			"Prefer direct workspace citations over inferred memory",
			"When sources disagree, surface disagreement explicitly",
			"Never present inference as confirmed assertion",
		},
		ForbiddenBehavior: []string{
			"Do not suggest write operations or hidden mutations in QA mode",
			"Do not fabricate code paths, owners, or run outcomes",
			"Do not leak unrelated data outside asked scope",
		},
		FallbackWhenUnknown: []string{
			"Return explicit unresolved reasons and next evidence needed",
			"Ask bounded follow-up questions when clarification is required",
			"Keep response deterministic and concise under low signal",
		},
	}),
}

const baselineSubagentsYAML = `agents:
  - id: domain-analyst
    skills: [service-inventory, interface-extraction, integration-mapping, datastore-mapping, cicd-mapping, ownership-coverage]
  - id: architect-aggregator
    skills: [findings, proposals]
  - id: system-analyst-qa
    skills: [qa]
`

func BaselineSubagentsContent() []byte {
	return []byte(baselineSubagentsYAML)
}

func (r Root) EnsureBaselineBundle() error {
	if err := r.writeFileIfMissing("skills/subagents.yaml", BaselineSubagentsContent()); err != nil {
		return err
	}

	for _, skill := range baselineSkillIDs {
		manifest := fmt.Sprintf("name: %s\nversion: 1\napplies_to: [mvp]\ninputs: []\noutputs: []\n", skill)
		if err := r.writeFileIfMissing(fmt.Sprintf("skills/%s/manifest.yaml", skill), []byte(manifest)); err != nil {
			return err
		}
		if err := r.writeFileIfMissing(
			fmt.Sprintf("skills/%s/prompts/system.md", skill),
			[]byte(renderSkillSystemPrompt(skill)),
		); err != nil {
			return err
		}
		if err := r.writeFileIfMissing(
			fmt.Sprintf("skills/%s/prompts/task.md", skill),
			[]byte(renderSkillTaskPrompt(skill)),
		); err != nil {
			return err
		}
		if err := r.writeFileIfMissing(fmt.Sprintf("skills/%s/templates/adr.md", skill), []byte(defaultADRTemplate)); err != nil {
			return err
		}
		if err := r.writeFileIfMissing(fmt.Sprintf("skills/%s/templates/rfc.md", skill), []byte(defaultRFCTemplate)); err != nil {
			return err
		}
	}

	for pack, content := range baselinePromptPacks {
		if err := r.writeFileIfMissing(fmt.Sprintf("skills/prompt-packs/%s.md", pack), []byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func (r Root) writeFileIfMissing(relPath string, content []byte) error {
	abs, err := r.Resolve(relPath)
	if err != nil {
		return err
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		if info.IsDir() {
			return fmt.Errorf("baseline seed target %q is a directory", relPath)
		}
		return nil
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("stat baseline seed target %q: %w", relPath, statErr)
	}
	return r.WriteFile(relPath, content)
}

func renderSkillSystemPrompt(skill string) string {
	spec := skillPromptSpecFor(skill)
	return renderStructuredPrompt(
		fmt.Sprintf("Skill System Prompt: %s", skill),
		structuredPromptSpec{
			Goal: fmt.Sprintf("Act as ACP skill %q. %s", skill, spec.Goal),
			Inputs: append([]string{
				"task meta (step_id, repo scopes, path scopes)",
				"workspace model/reports/charter artifacts relevant to this skill",
			}, spec.Inputs...),
			RequiredOutputShape: append([]string{
				"Deterministic, reviewable outputs that fit TaskResult contract expectations",
				"Stable IDs, canonical enums, and explicit provenance evidence arrays",
			}, spec.RequiredOutputShape...),
			EvidencePolicy: append([]string{
				"Evidence-first extraction from repository paths and docs",
				"Explicit uncertainty via questions/coverage when evidence is incomplete",
			}, spec.EvidencePolicy...),
			ForbiddenBehavior: append([]string{
				"No fabricated entities, integrations, owners, or deployment assumptions",
				"No markdown wrappers around JSON payloads expected by runtime parser",
			}, spec.ForbiddenBehavior...),
			FallbackWhenUnknown: append([]string{
				"Prefer partial valid output over invalid or speculative output",
				"Escalate unknowns in deterministic wording",
			}, spec.FallbackWhenUnknown...),
		},
	)
}

func renderSkillTaskPrompt(skill string) string {
	spec := skillPromptSpecFor(skill)
	return renderStructuredPrompt(
		fmt.Sprintf("Skill Task Prompt: %s", skill),
		structuredPromptSpec{
			Goal: fmt.Sprintf("Execute %q on current runtime shard with deterministic ACP conventions. %s", skill, spec.Goal),
			Inputs: append([]string{
				"assigned repo scope and optional shard path scopes",
				"current step objective from orchestrator",
			}, spec.Inputs...),
			RequiredOutputShape: append([]string{
				"TaskResult-safe contributions for entities/edges/findings/questions/coverage",
				"Actionable summaries without non-deterministic filler text",
			}, spec.RequiredOutputShape...),
			EvidencePolicy: append([]string{
				"Cite concrete file paths for observations whenever possible",
				"Use inference only with confidence and rationale",
			}, spec.EvidencePolicy...),
			ForbiddenBehavior: append([]string{
				"Do not return empty placeholder JSON when meaningful signal exists",
				"Do not mutate unrelated domains outside assigned scope",
			}, spec.ForbiddenBehavior...),
			FallbackWhenUnknown: append([]string{
				"Emit canonical coverage gaps and prioritized questions",
				"Keep payload syntactically valid even with sparse evidence",
			}, spec.FallbackWhenUnknown...),
		},
	)
}

type structuredPromptSpec struct {
	Goal                string
	Inputs              []string
	RequiredOutputShape []string
	EvidencePolicy      []string
	ForbiddenBehavior   []string
	FallbackWhenUnknown []string
}

type skillPromptSpec struct {
	Goal                string
	Inputs              []string
	RequiredOutputShape []string
	EvidencePolicy      []string
	ForbiddenBehavior   []string
	FallbackWhenUnknown []string
}

var baselineSkillPromptSpecs = map[string]skillPromptSpec{
	"service-inventory": {
		Goal: "Identify runtime services and core responsibilities with deterministic IDs.",
		Inputs: []string{
			"service entrypoints, module boundaries, deployment descriptors",
		},
		RequiredOutputShape: []string{
			"service entities with owner hints and provenance evidence",
		},
		EvidencePolicy: []string{
			"Prefer manifests, bootstrap files, and routing config as evidence",
		},
		ForbiddenBehavior: []string{
			"Do not collapse distinct services into one inferred monolith without proof",
		},
		FallbackWhenUnknown: []string{
			"Emit owner/ownership unknowns explicitly via coverage/questions",
		},
	},
	"interface-extraction": {
		Goal: "Extract API/event interfaces and attach them to owning services.",
		Inputs: []string{
			"router specs, protobuf/openapi files, message schemas",
		},
		RequiredOutputShape: []string{
			"api.* entities and relation edges with stable naming",
		},
		EvidencePolicy: []string{
			"Interface definitions require direct source-path evidence",
		},
		ForbiddenBehavior: []string{
			"Do not infer request/response contracts from naming alone",
		},
		FallbackWhenUnknown: []string{
			"Ask focused contract questions when definitions are partial",
		},
	},
	"integration-mapping": {
		Goal: "Map outbound and inbound integrations between services and external systems.",
		Inputs: []string{
			"HTTP clients, SDK configs, queue bindings, webhooks",
		},
		RequiredOutputShape: []string{
			"external.system entities and integration edges with direction",
		},
		EvidencePolicy: []string{
			"Integration claims require concrete call/config evidence",
		},
		ForbiddenBehavior: []string{
			"Do not treat commented or dead-code integrations as active without explicit runtime hints",
		},
		FallbackWhenUnknown: []string{
			"Emit dependency graph gaps for missing integration signal",
		},
	},
	"datastore-mapping": {
		Goal: "Identify datastore dependencies and service-to-datastore bindings.",
		Inputs: []string{
			"migration folders, ORM configs, SQL/noSQL clients, infra manifests",
		},
		RequiredOutputShape: []string{
			"datastore entities and binding edges with evidence-backed paths",
		},
		EvidencePolicy: []string{
			"Prefer connection settings and migration evidence over naming heuristics",
		},
		ForbiddenBehavior: []string{
			"Do not infer datastore technology from package names only",
		},
		FallbackWhenUnknown: []string{
			"Add datastore-binding unknowns to coverage.missing",
		},
	},
	"cicd-mapping": {
		Goal: "Capture CI/CD pipelines, deployment patterns, and delivery evidence.",
		Inputs: []string{
			"CI workflow files, deployment scripts, helm/manifests",
		},
		RequiredOutputShape: []string{
			"coverage notes/questions and optional findings-ready evidence references",
		},
		EvidencePolicy: []string{
			"Treat pipeline configuration files as primary CI/CD evidence",
		},
		ForbiddenBehavior: []string{
			"Do not claim release automation if only local scripts exist",
		},
		FallbackWhenUnknown: []string{
			"Emit ci-cd evidence gap questions with explicit missing artifacts",
		},
	},
	"ownership-coverage": {
		Goal: "Resolve ownership coverage and highlight unknown owner bindings.",
		Inputs: []string{
			"charter team cards, CODEOWNERS, repo docs, commit metadata hints",
		},
		RequiredOutputShape: []string{
			"owner_team_id assignments when proven, otherwise explicit questions",
		},
		EvidencePolicy: []string{
			"Owner assignment requires explicit mapping evidence",
		},
		ForbiddenBehavior: []string{
			"Do not auto-create teams from weak textual hints",
		},
		FallbackWhenUnknown: []string{
			"Record owner gaps in coverage with high-priority questions",
		},
	},
	"findings": {
		Goal: "Synthesize high-signal architecture risks and anti-pattern findings.",
		Inputs: []string{
			"model graph, coverage gaps, charter rules",
		},
		RequiredOutputShape: []string{
			"add_finding operations with deterministic IDs and severity",
		},
		EvidencePolicy: []string{
			"Each finding must link to evidence or transparent inference",
		},
		ForbiddenBehavior: []string{
			"Do not emit duplicate findings with paraphrased titles",
		},
		FallbackWhenUnknown: []string{
			"Promote uncertain risks to questions rather than hard findings",
		},
	},
	"proposals": {
		Goal: "Generate migration-oriented proposals grounded in validated findings.",
		Inputs: []string{
			"findings set, charter constraints, proposal templates",
		},
		RequiredOutputShape: []string{
			"deterministic proposal outline with rollout and risk sections",
		},
		EvidencePolicy: []string{
			"Proposal scope must map back to one or more findings",
		},
		ForbiddenBehavior: []string{
			"Do not prescribe broad rewrites without supporting findings",
		},
		FallbackWhenUnknown: []string{
			"Use phased TODO markers for unknown owners or effort estimates",
		},
	},
	"qa": {
		Goal: "Answer architecture questions from workspace artifacts in read-only mode.",
		Inputs: []string{
			"user question, model/reports/cards context, imports docs",
		},
		RequiredOutputShape: []string{
			"answer + citations + unresolved reasoning where needed",
		},
		EvidencePolicy: []string{
			"Prefer direct citations from workspace files",
		},
		ForbiddenBehavior: []string{
			"Do not propose write operations while answering read-only QA",
		},
		FallbackWhenUnknown: []string{
			"Return unresolved explanation and required evidence list",
		},
	},
}

func skillPromptSpecFor(skill string) skillPromptSpec {
	spec, ok := baselineSkillPromptSpecs[skill]
	if ok {
		return spec
	}
	return skillPromptSpec{
		Goal: "Produce deterministic, evidence-backed outputs aligned with ACP runtime contracts.",
		Inputs: []string{
			"workspace artifacts relevant to this skill",
		},
		RequiredOutputShape: []string{
			"TaskResult-compatible operations and structured unknown handling",
		},
		EvidencePolicy: []string{
			"Do not assert facts without evidence paths",
		},
		ForbiddenBehavior: []string{
			"Do not invent domain facts or ownership mappings",
		},
		FallbackWhenUnknown: []string{
			"Emit questions and coverage gaps using canonical terms",
		},
	}
}

func renderStructuredPrompt(title string, spec structuredPromptSpec) string {
	buf := bytes.Buffer{}
	buf.WriteString("# ")
	buf.WriteString(strings.TrimSpace(title))
	buf.WriteString("\n\n")

	sections := []struct {
		heading string
		lines   []string
	}{
		{heading: "Goal", lines: []string{strings.TrimSpace(spec.Goal)}},
		{heading: "Inputs", lines: spec.Inputs},
		{heading: "Required Output Shape", lines: spec.RequiredOutputShape},
		{heading: "Evidence Policy", lines: spec.EvidencePolicy},
		{heading: "Forbidden Behavior", lines: spec.ForbiddenBehavior},
		{heading: "Fallback When Unknown", lines: spec.FallbackWhenUnknown},
	}

	for _, section := range sections {
		buf.WriteString("## ")
		buf.WriteString(section.heading)
		buf.WriteString("\n\n")
		lines := normalizePromptLines(section.lines)
		if len(lines) == 0 {
			buf.WriteString("- none\n\n")
			continue
		}
		for _, line := range lines {
			buf.WriteString("- ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func normalizePromptLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

const defaultADRTemplate = `# ADR

## Context

## Decision

## Consequences
`

const defaultRFCTemplate = `# RFC

## Summary

## Motivation

## Detailed Design

## Rollout Plan
`
