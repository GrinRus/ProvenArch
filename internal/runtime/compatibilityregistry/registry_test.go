package compatibilityregistry

import "testing"

func TestRulesEnumerateOnlyActiveRepairSurface(t *testing.T) {
	t.Parallel()

	rules := Rules()
	if len(rules) != 2 {
		t.Fatalf("expected exactly 2 active repair rules, got %d", len(rules))
	}

	required := []string{
		RuleSafeCollectDocumentPathNormalization,
		RuleDraftRootReconcileExistingOutputs,
	}
	for _, id := range required {
		rule, ok := RuleForID(id)
		if !ok {
			t.Fatalf("expected repair rule %q to exist", id)
		}
		if rule.Trigger == "" || rule.Preconditions == "" || rule.ExactMutation == "" || rule.FailureSemantics == "" || rule.RemovalTarget == "" {
			t.Fatalf("expected full metadata for repair rule %q, got %#v", id, rule)
		}
	}
}
