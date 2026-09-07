package orchestrator

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScenarioFixturesHaveTrackedGoldenSnapshots(t *testing.T) {
	t.Parallel()

	scenarios := []string{
		"single-service-http-postgres-gitlabci",
		"two-services-http-call-and-queue",
		"missing-owner-and-missing-cicd",
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			root := filepath.Join("..", "..", "fixtures", "scenarios", scenario)
			for _, rel := range []string{
				"workspace/workspace.yaml",
				"golden/README.md",
				"golden/snapshot.sha256",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
					t.Fatalf("missing scenario fixture %s: %v", filepath.ToSlash(filepath.Join(root, rel)), err)
				}
			}

			file, err := os.Open(filepath.Join(root, "golden", "snapshot.sha256"))
			if err != nil {
				t.Fatalf("open golden snapshot: %v", err)
			}
			defer file.Close()

			entries := 0
			readableEntries := 0
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) != 2 {
					t.Fatalf("invalid snapshot entry %q", line)
				}
				digest, decodeErr := hex.DecodeString(fields[0])
				if decodeErr != nil || len(digest) != sha256.Size {
					t.Fatalf("invalid sha256 digest in %q", line)
				}
				rel := filepath.FromSlash(fields[1])
				if filepath.IsAbs(rel) || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					t.Fatalf("snapshot path escapes fixture root: %q", fields[1])
				}
				goldenPath := filepath.Join(root, "golden", "readable", rel)
				content, readErr := os.ReadFile(goldenPath)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						// Machine-only snapshot entries are valid; the readable drift
						// checker owns the tracked-output coverage rule.
						entries++
						continue
					}
					t.Fatalf("read golden output %q: %v", fields[1], readErr)
				}
				readableEntries++
				actual := sha256.Sum256(content)
				if !strings.EqualFold(hex.EncodeToString(actual[:]), fields[0]) {
					t.Fatalf("golden digest mismatch for %q", fields[1])
				}
				entries++
			}
			if err := scanner.Err(); err != nil {
				t.Fatalf("scan golden snapshot: %v", err)
			}
			if entries == 0 {
				t.Fatal("golden snapshot has no entries")
			}
			if readableEntries == 0 {
				t.Fatal("golden snapshot has no readable entries")
			}
		})
	}
}
