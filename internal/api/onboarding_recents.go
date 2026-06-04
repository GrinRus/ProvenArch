package api

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const onboardingRecentWorkspaceLimit = 10

type onboardingRecentWorkspace struct {
	Path         string `json:"path"`
	LastOpenedAt string `json:"last_opened_at"`
	Exists       bool   `json:"exists"`
}

type onboardingRecentWorkspacesFile struct {
	Version    int                         `json:"version"`
	Workspaces []onboardingRecentWorkspace `json:"workspaces"`
}

var onboardingRecentWorkspacesPath = defaultOnboardingRecentWorkspacesPath

func defaultOnboardingRecentWorkspacesPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "provenarch", "recent-workspaces.json"), nil
}

func loadOnboardingRecentWorkspaces() []onboardingRecentWorkspace {
	path, err := onboardingRecentWorkspacesPath()
	if err != nil {
		return nil
	}
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return nil
	}
	var stored onboardingRecentWorkspacesFile
	if err := json.Unmarshal(payload, &stored); err != nil {
		return nil
	}
	workspaces := dedupeRecentWorkspaces(stored.Workspaces)
	for index := range workspaces {
		workspaces[index].Exists = workspaceDirectoryExists(workspaces[index].Path)
	}
	return workspaces
}

func recordOnboardingRecentWorkspace(workspacePath string, openedAt time.Time) error {
	normalized := strings.TrimSpace(filepath.Clean(workspacePath))
	if normalized == "" {
		return nil
	}
	workspaces := append([]onboardingRecentWorkspace{
		{
			Path:         normalized,
			LastOpenedAt: openedAt.UTC().Format(time.RFC3339),
			Exists:       workspaceDirectoryExists(normalized),
		},
	}, loadOnboardingRecentWorkspaces()...)
	return saveOnboardingRecentWorkspaces(dedupeRecentWorkspaces(workspaces))
}

func forgetOnboardingRecentWorkspace(workspacePath string) error {
	normalized := strings.TrimSpace(filepath.Clean(workspacePath))
	if normalized == "" {
		return nil
	}
	workspaces := loadOnboardingRecentWorkspaces()
	filtered := workspaces[:0]
	for _, item := range workspaces {
		if filepath.Clean(item.Path) == normalized {
			continue
		}
		filtered = append(filtered, item)
	}
	return saveOnboardingRecentWorkspaces(filtered)
}

func saveOnboardingRecentWorkspaces(workspaces []onboardingRecentWorkspace) error {
	path, err := onboardingRecentWorkspacesPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(onboardingRecentWorkspacesFile{
		Version:    1,
		Workspaces: dedupeRecentWorkspaces(workspaces),
	}, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func dedupeRecentWorkspaces(workspaces []onboardingRecentWorkspace) []onboardingRecentWorkspace {
	seen := map[string]bool{}
	deduped := make([]onboardingRecentWorkspace, 0, len(workspaces))
	for _, item := range workspaces {
		normalized := strings.TrimSpace(filepath.Clean(item.Path))
		if normalized == "." || normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		deduped = append(deduped, onboardingRecentWorkspace{
			Path:         normalized,
			LastOpenedAt: strings.TrimSpace(item.LastOpenedAt),
			Exists:       item.Exists,
		})
	}
	sort.SliceStable(deduped, func(left, right int) bool {
		return deduped[left].LastOpenedAt > deduped[right].LastOpenedAt
	})
	if len(deduped) > onboardingRecentWorkspaceLimit {
		deduped = deduped[:onboardingRecentWorkspaceLimit]
	}
	return deduped
}

func workspaceDirectoryExists(workspacePath string) bool {
	info, err := os.Stat(workspacePath)
	return err == nil && info.IsDir()
}
