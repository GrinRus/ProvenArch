package compatibilityregistry

type Rule struct {
	ID               string
	Trigger          string
	Preconditions    string
	ExactMutation    string
	FailureSemantics string
	RemovalTarget    string
}

const (
	RuleSafeCollectDocumentPathNormalization = "collect.documents_path_normalization"
	RuleDraftRootReconcileExistingOutputs    = "drafts.reconcile_existing_canonical_outputs"
)

var rules = []Rule{
	{
		ID:               RuleSafeCollectDocumentPathNormalization,
		Trigger:          "collect manifest document path points to write_root-available file but uses artifact_root-prefixed or absolute path",
		Preconditions:    "document file already exists under write_root and normalized path is unambiguous artifact-root-relative path",
		ExactMutation:    "rewrite documents[].path to artifact_root-relative form only",
		FailureSemantics: "if path is ambiguous or file is absent, keep path unchanged and let contract validation fail",
		RemovalTarget:    "remove when providers stop emitting workspace-relative or duplicated artifact-root document paths",
	},
	{
		ID:               RuleDraftRootReconcileExistingOutputs,
		Trigger:          "runtime draft manifest is structurally valid but referenced draft_root file is missing while same content already exists at canonical_path under draft_root",
		Preconditions:    "repair stage only; manifest is valid for task and canonical fallback points to an existing file inside draft_root",
		ExactMutation:    "copy canonical fallback file into the expected manifest-relative draft_root path",
		FailureSemantics: "if fallback file is absent or invalid, perform no mutation and let draft validation fail",
		RemovalTarget:    "remove when providers stop writing draft files only at canonical_path-shaped locations during repair",
	},
}

func Rules() []Rule {
	out := make([]Rule, len(rules))
	copy(out, rules)
	return out
}

func RuleForID(id string) (Rule, bool) {
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}
