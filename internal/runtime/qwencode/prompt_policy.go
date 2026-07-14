package qwencode

import (
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
)

func buildQwenArgsWithIncludeDirectories(includeDirs []string, prompt string) []string {
	return buildQwenArgsWithPermissions(includeDirs, prompt, acpruntime.DefaultPermissions())
}

func buildQwenArgsWithPermissions(includeDirs []string, prompt string, permissions acpruntime.PermissionValues) []string {
	args := []string{
		"--chat-recording",
		"false",
		"--channel",
		"CI",
		"--output-format",
		"stream-json",
		"--include-partial-messages",
	}
	if strings.TrimSpace(permissions.Mode) != acpruntime.PermissionModeManaged {
		args = append(args, "--yolo")
	}
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
