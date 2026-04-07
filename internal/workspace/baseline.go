package workspace

import (
	"fmt"
	"os"
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
	"constitution": `# Constitution Prompt Pack

Use the wizard contract and charter templates to produce deterministic baseline constitution artifacts.
Do not invent organization facts; preserve unresolved items explicitly.
`,
	"collect-context": `# Collect Context Prompt Pack

Analyze code, configs, and docs for the selected repo scopes.
Emit evidence-backed observations, and surface unknowns through questions and coverage gaps.
`,
	"findings": `# Findings Prompt Pack

Derive architecture findings from model and charter rules.
Prioritize evidence-backed risks and anti-patterns with deterministic wording.
`,
	"proposals": `# Proposals Prompt Pack

Compile implementation proposals from findings and model context.
Keep proposal structure deterministic and migration-oriented.
`,
	"qa": `# QA Prompt Pack

Answer read-only architecture questions using workspace artifacts and citations.
If evidence is insufficient, return unresolved reasons instead of guesses.
`,
}

const baselineSubagentsYAML = `agents:
  - id: domain-analyst
    skills: [service-inventory, interface-extraction, integration-mapping, datastore-mapping, cicd-mapping, ownership-coverage]
  - id: architect-aggregator
    skills: [findings, proposals]
  - id: system-analyst-qa
    skills: [qa]
`

func (r Root) EnsureBaselineBundle() error {
	if err := r.writeFileIfMissing("skills/subagents.yaml", []byte(baselineSubagentsYAML)); err != nil {
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
	return strings.TrimSpace(fmt.Sprintf(`
# System Prompt

You are ACP skill "%s".
Work evidence-first: reference concrete files and avoid unsupported assumptions.
When evidence is incomplete, emit questions and coverage gaps instead of guessing.
`, skill)) + "\n"
}

func renderSkillTaskPrompt(skill string) string {
	return strings.TrimSpace(fmt.Sprintf(`
# Task Prompt

Analyze the assigned repo scope for skill "%s".
Produce deterministic, reviewable outputs suitable for workspace materialization.
`, skill)) + "\n"
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
