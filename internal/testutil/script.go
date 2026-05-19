package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func WriteExecutableScript(t testing.TB, filename string, script string) string {
	t.Helper()

	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "runner-stub.sh"
	}
	path := filepath.Join(t.TempDir(), filename)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write executable script %q: %v", filename, err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		t.Fatalf("chmod executable script %q: %v", filename, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		t.Fatalf("publish executable script %q: %v", filename, err)
	}
	return path
}
