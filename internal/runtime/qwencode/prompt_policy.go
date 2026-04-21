package qwencode

import (
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
)

func buildDefaultQwenArgs(task acpruntime.Task, prompt string) []string {
	return buildQwenArgsWithIncludeDirectories(acpruntime.ResolveHeadlessIncludeDirectories(task), prompt)
}

func buildQwenArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	args := []string{"--output-format", "json", "--chat-recording", "false", "--yolo", "--channel", "CI"}
	for _, dir := range includeDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "-p", prompt)
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderQwenCode, task)
}
