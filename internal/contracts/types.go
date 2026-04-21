package contracts

type RuntimeMeta struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type Entity struct {
	ID          string     `json:"id" yaml:"id"`
	Type        string     `json:"type" yaml:"type"`
	Name        string     `json:"name" yaml:"name"`
	Aliases     []string   `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Tags        []string   `json:"tags,omitempty" yaml:"tags,omitempty"`
	Attributes  any        `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	OwnerTeamID string     `json:"owner_team_id,omitempty" yaml:"owner_team_id,omitempty"`
	Provenance  Provenance `json:"provenance" yaml:"provenance"`
}

type Edge struct {
	ID         string     `json:"id" yaml:"id"`
	Type       string     `json:"type" yaml:"type"`
	From       string     `json:"from" yaml:"from"`
	To         string     `json:"to" yaml:"to"`
	Name       string     `json:"name,omitempty" yaml:"name,omitempty"`
	Attributes any        `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Provenance Provenance `json:"provenance" yaml:"provenance"`
}

type Finding struct {
	ID          string     `json:"id"`
	Severity    string     `json:"severity"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	RuleID      string     `json:"rule_id,omitempty"`
	RelatedIDs  []string   `json:"related_ids,omitempty"`
	Provenance  Provenance `json:"provenance"`
}

type Question struct {
	ID         string   `json:"id"`
	Text       string   `json:"text"`
	Priority   string   `json:"priority,omitempty"`
	RelatedIDs []string `json:"related_ids,omitempty"`
}

type Coverage struct {
	Observed []string `json:"observed,omitempty"`
	Missing  []string `json:"missing,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

type Provenance struct {
	Kind       string     `json:"kind" yaml:"kind"`
	Confidence float64    `json:"confidence" yaml:"confidence"`
	Evidence   []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type Evidence struct {
	Repo        string     `json:"repo" yaml:"repo"`
	Ref         string     `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path        string     `json:"path" yaml:"path"`
	Lines       *LineRange `json:"lines,omitempty" yaml:"lines,omitempty"`
	ExcerptHash string     `json:"excerpt_hash,omitempty" yaml:"excerpt_hash,omitempty"`
	Excerpt     string     `json:"excerpt,omitempty" yaml:"excerpt,omitempty"`
}

type LineRange struct {
	Start int `json:"start" yaml:"start"`
	End   int `json:"end" yaml:"end"`
}

type DocArtifact struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Format     string   `json:"format,omitempty"`
	RelatedIDs []string `json:"related_ids,omitempty"`
}

type SemanticSnapshot struct {
	Coverage  Coverage   `json:"coverage"`
	Questions []Question `json:"questions"`
	Entities  []Entity   `json:"entities"`
	Edges     []Edge     `json:"edges"`
	Findings  []Finding  `json:"findings"`
}
