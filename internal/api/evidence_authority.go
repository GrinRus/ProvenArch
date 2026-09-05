package api

const (
	evidenceAuthorityPromotedCurrent = "promoted_current"

	evidenceAuthorityQASnapshot = "qa_snapshot"
	evidenceAuthorityQAAudit    = "qa_audit"
)

type evidenceAuthority struct {
	Mode  string `json:"mode"`
	RunID string `json:"run_id,omitempty"`
	Root  string `json:"root"`
}
