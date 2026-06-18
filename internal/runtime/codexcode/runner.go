package codexcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/promptcontract"
	"github.com/GrinRus/ProvenArch/internal/runtime/providercommon"
)

var ErrRunnerUnavailable = errors.New("codex-code runner is unavailable")

type HeadlessRunner struct {
	Command string
	Args    []string
}

func (r HeadlessRunner) RuntimeMeta() contracts.RuntimeMeta {
	return contracts.RuntimeMeta{Name: string(acpruntime.ProviderCodexCode), Version: "headless"}
}

func (r HeadlessRunner) commandName() string {
	command := strings.TrimSpace(r.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv("ACP_CODEX_CMD"))
	}
	if command == "" {
		command = "codex"
	}
	return command
}

func (r HeadlessRunner) Preflight(_ context.Context) error {
	return providercommon.PreflightCommand(acpruntime.ProviderCodexCode, r.commandName())
}

func (r HeadlessRunner) Run(ctx context.Context, task acpruntime.Task) (acpruntime.Result, error) {
	if err := r.Preflight(ctx); err != nil {
		return acpruntime.Result{}, err
	}
	return providercommon.RunHeadlessProvider(ctx, task, codexAdapter{runner: r})
}

type codexAdapter struct {
	runner HeadlessRunner
}

func (a codexAdapter) Provider() acpruntime.Provider {
	return acpruntime.ProviderCodexCode
}

func (a codexAdapter) RuntimeVersion() string {
	return "headless"
}

func (a codexAdapter) CommandSpec(task acpruntime.Task) (providercommon.CommandSpec, error) {
	includeDirs := acpruntime.ResolveHeadlessIncludeDirectories(task)
	return a.commandSpecWithPrompt(task, includeDirs, buildPrompt(task))
}

func (a codexAdapter) commandSpecWithPrompt(task acpruntime.Task, includeDirs []string, prompt string) (providercommon.CommandSpec, error) {
	cwd := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task))
	commandArgs := append([]string(nil), a.runner.Args...)
	if len(commandArgs) == 0 {
		commandArgs = buildCodexArgsWithPermissions(cwd, includeDirs, task.RuntimePermissions)
	} else if strings.TrimSpace(task.RuntimePermissions.Mode) == acpruntime.PermissionModeManaged {
		commandArgs = stripCodexDangerFullAccessArgs(commandArgs)
	}
	env, err := a.commandEnv(task)
	if err != nil {
		return providercommon.CommandSpec{}, err
	}
	return providercommon.CommandSpec{
		Provider:    acpruntime.ProviderCodexCode,
		Command:     a.runner.commandName(),
		Args:        commandArgs,
		Env:         env,
		Stdin:       strings.NewReader(prompt),
		Dir:         cwd,
		PromptBytes: len([]byte(prompt)),
		IncludeDirs: includeDirs,
	}, nil
}

func (a codexAdapter) commandEnv(task acpruntime.Task) (map[string]string, error) {
	if len(a.runner.Args) > 0 {
		return nil, nil
	}
	codexHome, err := prepareIsolatedCodexHome(task)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(codexHome) == "" {
		return nil, nil
	}
	return map[string]string{"CODEX_HOME": codexHome}, nil
}

func stripCodexDangerFullAccessArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		trimmed := strings.TrimSpace(args[i])
		if strings.HasPrefix(trimmed, "--sandbox=") && strings.TrimSpace(strings.TrimPrefix(trimmed, "--sandbox=")) == "danger-full-access" {
			continue
		}
		if trimmed == "--sandbox" && i+1 < len(args) && strings.TrimSpace(args[i+1]) == "danger-full-access" {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func (a codexAdapter) CollectManifestRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderCodexCode, providercommon.FocusedRepairCollectManifest, validationErr, a.commandSpecWithPrompt)
}

func (a codexAdapter) CollectArtifactPairRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderCodexCode, providercommon.FocusedRepairCollectArtifactPair, validationErr, a.commandSpecWithPrompt)
}

func (a codexAdapter) ValidatorVerdictRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderCodexCode, providercommon.FocusedRepairValidatorVerdict, validationErr, a.commandSpecWithPrompt)
}

func (a codexAdapter) DraftArtifactRepairCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderCodexCode, providercommon.FocusedRepairDraftArtifacts, validationErr, a.commandSpecWithPrompt)
}

func (a codexAdapter) DraftArtifactEnrichmentCommandSpec(task acpruntime.Task, validationErr error) (providercommon.CommandSpec, error) {
	return providercommon.BuildFocusedRepairCommandSpec(task, acpruntime.ProviderCodexCode, providercommon.FocusedRepairDraftEnrichment, validationErr, a.commandSpecWithPrompt)
}

func (a codexAdapter) ValidateArtifacts(task acpruntime.Task) error {
	return providercommon.ValidateRuntimeArtifacts(task, acpruntime.ProviderCodexCode)
}

func (a codexAdapter) ActivityPolicy(task acpruntime.Task) providercommon.ActivityPolicy {
	monitorArtifacts := providercommon.MonitorsRuntimeArtifacts(task)
	return providercommon.WithCollectArtifactEnrichmentWindow(task, providercommon.ActivityPolicy{
		MonitorArtifacts:   monitorArtifacts,
		MonitorPreArtifact: monitorArtifacts,
	})
}

func (a codexAdapter) RecoveryPolicy(_ acpruntime.Task) providercommon.RecoveryPolicy {
	return providercommon.RecoveryPolicy{
		AcceptValidArtifactsAfterStop:     true,
		RepairCollectManifestOnce:         true,
		RepairCollectArtifactPairOnce:     true,
		RepairValidatorVerdictOnce:        true,
		RepairDraftArtifactsOnce:          true,
		RepairDraftArtifactEnrichmentOnce: true,
	}
}

func (a codexAdapter) UnavailableMarkers() []string {
	return providercommon.DefaultUnavailableMarkers()
}

func buildDefaultCodexArgs(task acpruntime.Task) []string {
	cwd := strings.TrimSpace(acpruntime.ResolveHeadlessWorkingDirectory(task))
	return buildCodexArgsWithIncludeDirectories(cwd, acpruntime.ResolveHeadlessIncludeDirectories(task))
}

func buildCodexArgsWithIncludeDirectories(cwd string, includeDirs []string) []string {
	return buildCodexArgsWithPermissions(cwd, includeDirs, acpruntime.DefaultPermissions())
}

func buildCodexArgsWithPermissions(cwd string, includeDirs []string, permissions acpruntime.PermissionValues) []string {
	args := []string{
		"exec",
		"--json",
		"--color", "never",
		"--disable", "plugins",
		"--disable", "remote_plugin",
		"--disable", "plugin_sharing",
		"--disable", "apps",
		"--disable", "enable_mcp_apps",
		"--disable", "tool_suggest",
		"--disable", "skill_mcp_dependency_install",
		"--ignore-user-config",
		"--ignore-rules",
		"--skip-git-repo-check",
	}
	if strings.TrimSpace(permissions.Mode) == acpruntime.PermissionModeManaged {
		args = append(args, "--sandbox", "workspace-write")
	} else {
		args = append(args, "--sandbox", "danger-full-access")
	}
	if cwd != "" {
		args = append(args, "--cd", cwd)
	}
	for _, dir := range includeDirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "--ephemeral", "-")
	return args
}

func buildPrompt(task acpruntime.Task) string {
	return promptcontract.ComposeArtifactOnlyPrompt(acpruntime.ProviderCodexCode, task)
}

func prepareIsolatedCodexHome(task acpruntime.Task) (string, error) {
	sourceHome, err := sourceCodexHome()
	if err != nil {
		return "", err
	}
	base := strings.TrimSpace(os.Getenv("ACP_CODEX_ISOLATED_HOME_BASE"))
	if base == "" {
		base = filepath.Join(os.TempDir(), "provenarch-codex-home")
	}
	home := filepath.Join(filepath.Clean(base), safeCodexHomePart(task.RunID), safeCodexHomePart(firstNonEmpty(task.TaskID, task.StepID, "task")))
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", fmt.Errorf("create isolated codex home: %w", err)
	}
	for _, name := range []string{"auth.json", "installation_id"} {
		src := filepath.Join(sourceHome, name)
		if _, err := os.Stat(src); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("stat codex home file %s: %w", name, err)
		}
		mode := os.FileMode(0o600)
		if name != "auth.json" {
			mode = 0o644
		}
		if err := copyRegularFile(src, filepath.Join(home, name), mode); err != nil {
			return "", fmt.Errorf("copy codex home file %s: %w", name, err)
		}
	}
	return home, nil
}

func sourceCodexHome() (string, error) {
	if value := strings.TrimSpace(os.Getenv("CODEX_HOME")); value != "" {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home for codex auth: %w", err)
	}
	return filepath.Join(home, ".codex"), nil
}

func copyRegularFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Chmod(tmp, mode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

func safeCodexHomePart(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		return "default"
	}
	if len(out) > 120 {
		return out[:120]
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
