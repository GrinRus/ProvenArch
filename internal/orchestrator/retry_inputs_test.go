package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/workspace"
)

func TestRetryStagingReuseInvalidatesRequestedAndDownstreamScopes(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		step   string
		scopes []string
		want   bool
	}{
		{name: "charter reuses nothing", path: "shards/payments/manifest.json", step: "init.step0.constitution", want: false},
		{name: "full collect reuses nothing", path: "shards/users/manifest.json", step: "init.step1.collect", want: false},
		{name: "collect keeps validated sibling", path: "shards/users/manifest.json", step: "init.step1.collect", scopes: []string{"payments"}, want: true},
		{name: "collect invalidates failed scope", path: "shards/payments/manifest.json", step: "init.step1.collect", scopes: []string{"payments"}, want: false},
		{name: "as-is keeps collect", path: "shards/users/manifest.json", step: "init.step2.asis_docs", want: true},
		{name: "as-is invalidates final", path: "final/reports/as-is/overview.md", step: "init.step2.asis_docs", want: false},
		{name: "findings keeps as-is", path: "final/reports/as-is/overview.md", step: "init.step3.findings", want: true},
		{name: "findings invalidates gaps", path: "final/reports/coverage/summary.md", step: "init.step3.findings", want: false},
		{name: "proposals keeps findings", path: "final/reports/findings/findings.md", step: "init.step4.proposals", want: true},
		{name: "proposals invalidates proposal", path: "final/proposals/p-1/proposal.md", step: "init.step4.proposals", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryStagingPathReusable(test.path, test.step, test.scopes); got != test.want {
				t.Fatalf("retryStagingPathReusable(%q, %q) = %v, want %v", test.path, test.step, got, test.want)
			}
		})
	}
}

func TestCopyRetryStagingReusesSiblingWithoutMutatingParent(t *testing.T) {
	ws := workspace.Root{Path: t.TempDir()}
	parentPayment := "reports/taskruns/parent/staging/shards/payments/manifest.json"
	parentUsers := "reports/taskruns/parent/staging/shards/users/manifest.json"
	if err := ws.WriteFile(parentPayment, []byte("payments-parent")); err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile(parentUsers, []byte("users-parent")); err != nil {
		t.Fatal(err)
	}
	if err := copyRetryStaging(ws, "parent", "child", "init.step1.collect", []string{"payments"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "reports/taskruns/child/staging/shards/payments/manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("failed scope was reused: %v", err)
	}
	childUsers, err := os.ReadFile(filepath.Join(ws.Path, "reports/taskruns/child/staging/shards/users/manifest.json"))
	if err != nil || string(childUsers) != "users-parent" {
		t.Fatalf("validated sibling was not copied: %q, %v", childUsers, err)
	}
	parentUsersBytes, err := os.ReadFile(filepath.Join(ws.Path, filepath.FromSlash(parentUsers)))
	if err != nil || string(parentUsersBytes) != "users-parent" {
		t.Fatalf("immutable parent changed: %q, %v", parentUsersBytes, err)
	}
}
