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
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write executable script %q: %v", filename, err)
	}
	return path
}
