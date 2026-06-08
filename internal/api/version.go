package api

import (
	"strings"
	"sync"
)

type buildInfo struct {
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Built    string `json:"built"`
	UIBundle string `json:"ui_bundle"`
}

var currentBuildInfo = buildInfo{
	Version:  "dev",
	Commit:   "none",
	Built:    "unknown",
	UIBundle: "embedded",
}
var buildInfoMu sync.RWMutex

func SetBuildInfo(version, commit, built string) {
	buildInfoMu.Lock()
	defer buildInfoMu.Unlock()
	currentBuildInfo = buildInfo{
		Version:  firstNonEmpty(version, "dev"),
		Commit:   firstNonEmpty(commit, "none"),
		Built:    firstNonEmpty(built, "unknown"),
		UIBundle: "embedded",
	}
}

func CurrentBuildInfo() buildInfo {
	buildInfoMu.RLock()
	defer buildInfoMu.RUnlock()
	return currentBuildInfo
}

func firstNonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
