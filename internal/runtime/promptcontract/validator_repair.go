package promptcontract

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/artifactquality"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/steppolicy"
)

func validatorVerdictRepairWriteCommand(writeRoot string, target string, skeleton string) string {
	return strings.Join([]string{
		"mkdir -p " + shellSingleQuote(strings.TrimSpace(writeRoot)),
		"cat > " + shellSingleQuote(strings.TrimSpace(target)) + " <<'ACP_VALIDATOR_VERDICT_JSON'",
		strings.TrimSpace(skeleton),
		"ACP_VALIDATOR_VERDICT_JSON",
	}, "\n")
}

func ComposeValidatorVerdictRepairPrompt(provider acpruntime.Provider, task acpruntime.Task, validationErr error) string {
	target := filepath.Join(strings.TrimSpace(task.WriteRoot), "validator-verdict.json")
	skeleton := steppolicy.ValidatorVerdictTaskSkeleton(task)
	lines := []string{
		fmt.Sprintf("You are ACP runtime provider %q in validator verdict focused recovery mode.", provider),
		"Immediate validator verdict repair action:",
		"- Run the exact shell command below as your next command. Do not inspect repository files first.",
		fmt.Sprintf("- Write exactly one file now: %q.", target),
		"- Do not browse for more evidence before this write. The embedded skeleton is the first valid artifact set.",
		"- If validator-verdict.json already exists but is invalid, overwrite it from the heredoc command.",
		"- Copy the heredoc JSON exactly first. Do not make factual edits before the verdict validates.",
		"VALIDATOR VERDICT WRITE COMMAND:",
		validatorVerdictRepairWriteCommand(task.WriteRoot, target, skeleton),
		"Artifact-only recovery contract:",
		"- Do not return semantic JSON or any semantic payload on stdout.",
		"- Do not write shard-pack-manifest.json, draft manifests, markdown reports, raw logs, or sibling taskrun files.",
		"- Final action must be: write only write_root/validator-verdict.json, then exit successfully.",
		"- Exit with code 0 only after validator-verdict.json validates.",
		fmt.Sprintf(`- write_root (absolute) = %q`, strings.TrimSpace(task.WriteRoot)),
		fmt.Sprintf(`- validator-verdict.json absolute target = %q`, target),
		fmt.Sprintf(`- read_context_roots = %q`, strings.Join(task.ReadContextRoots, ", ")),
		"VALIDATOR VERDICT JSON SKELETON:",
		skeleton,
		"VALIDATOR VERDICT REPAIR INSTRUCTIONS:",
		"- The heredoc JSON is the complete first repair artifact; write it first, then exit if it validates.",
		"- Do not inspect sibling baseline workspaces, prior reports/taskruns history, or raw provider logs as examples.",
		"- If you do inspect staged final artifacts after writing the first valid artifact set, adjust only checked_paths, verdict, summary, fixed_paths, findings, questions, and issues.",
		"- If the only residual gap is missing owner mapping evidence, keep it in findings/questions and use verdict=PASS when there are no technical validator issues.",
		"- If there is a technical validator issue, use verdict=FAIL and encode it only in canonical issues[] shape.",
		"- issues[] items must use only: code, severity, message, path, document_id, citation_id.",
		"- issues[].severity must be error or warning only.",
		"- Legacy issue fields are forbidden inside issues[]: id, title, description, rule_id, related_paths, related_ids, provenance.",
		"VALIDATOR VERDICT CANONICAL SHAPE:",
	}
	lines = append(lines, artifactquality.ValidatorVerdictContractLines()...)
	lines = append(lines,
		"- Canonical fragment below is normative for validator metadata and finding evidence shape; copy keys/types exactly and only change IDs/content.",
		artifactquality.ValidatorVerdictCanonicalExample(),
		"- Canonical issues[] item example:",
		artifactquality.ValidatorVerdictIssueCanonicalExample(),
	)
	if detail := errorText(validationErr); detail != "" {
		lines = append(lines, fmt.Sprintf("- Previous validator artifact validation failure: %s", detail))
	}
	return strings.Join(lines, "\n")
}
