package validation

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	schemadata "github.com/GrinRus/ProvenArch/schemas"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Schema string

const (
	WorkspaceSchema         Schema = "workspace.schema.json"
	ShardPackManifestSchema Schema = "shard-pack-manifest.schema.json"
	FinalRunIndexSchema     Schema = "final-run-index.schema.json"
	CitationIndexSchema     Schema = "citation-index.schema.json"
	ValidatorVerdictSchema  Schema = "validator-verdict.schema.json"
	QAAnswerSchema          Schema = "qa-answer.schema.json"
	SourceRevisionsSchema   Schema = "source-revisions.schema.json"
	RefreshImpactPlanSchema Schema = "refresh-impact-plan.schema.json"
)

var (
	compiledSchemas sync.Map
)

func ValidateRawJSON(schemaName Schema, raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("%s validation failed: empty payload", schemaName)
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("%s validation failed: decode json: %w", schemaName, err)
	}
	return ValidatePayload(schemaName, payload)
}

func ValidatePayload(schemaName Schema, payload any) error {
	schema, err := compileSchema(schemaName)
	if err != nil {
		return err
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("%s validation failed: %w", schemaName, err)
	}
	return nil
}

func compileSchema(schemaName Schema) (*jsonschema.Schema, error) {
	key := string(schemaName)
	if cached, ok := compiledSchemas.Load(key); ok {
		schema, _ := cached.(*jsonschema.Schema)
		if schema != nil {
			return schema, nil
		}
	}

	schemaBytes, err := schemadata.Load(key)
	if err != nil {
		return nil, err
	}

	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true
	resource := "https://example.local/acp/schemas/" + strings.TrimSpace(key)
	if err := compiler.AddResource(resource, strings.NewReader(string(schemaBytes))); err != nil {
		return nil, fmt.Errorf("compile %s: add resource: %w", schemaName, err)
	}

	schema, err := compiler.Compile(resource)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", schemaName, err)
	}

	compiledSchemas.Store(key, schema)
	return schema, nil
}
