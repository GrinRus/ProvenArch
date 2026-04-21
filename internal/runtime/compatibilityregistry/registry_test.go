package compatibilityregistry

import "testing"

func TestRulesEnumerateOnlyActiveCompatibilitySurface(t *testing.T) {
	t.Parallel()

	rules := Rules()
	if len(rules) != 3 {
		t.Fatalf("expected exactly 3 active compatibility rules, got %d", len(rules))
	}

	required := []string{
		RuleSafeCollectDocumentPathNormalization,
		RuleDropDuplicateLegacyAddDocArtifact,
		RuleDraftRootReconcileExistingOutputs,
	}
	for _, id := range required {
		rule, ok := RuleForID(id)
		if !ok {
			t.Fatalf("expected compatibility rule %q to exist", id)
		}
		if rule.Trigger == "" || rule.Preconditions == "" || rule.ExactMutation == "" || rule.FailureSemantics == "" || rule.RemovalTarget == "" {
			t.Fatalf("expected full metadata for compatibility rule %q, got %#v", id, rule)
		}
	}
}
