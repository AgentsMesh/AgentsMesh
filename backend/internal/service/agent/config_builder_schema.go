package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/parser"
)

// GetConfigSchema returns the config schema for an agent.
// For PodFile agents, CONFIG declarations are extracted from the PodFile source.
func (b *ConfigBuilder) GetConfigSchema(ctx context.Context, agentSlug string) (*ConfigSchemaResponse, error) {
	agentDef, err := b.provider.GetAgent(ctx, agentSlug)
	if err != nil {
		return nil, err
	}
	if agentDef.PodfileSource != nil && *agentDef.PodfileSource != "" {
		return b.getConfigSchemaFromPodFile(*agentDef.PodfileSource)
	}
	return b.buildConfigSchemaResponse(&agentDef.ConfigSchema), nil
}

// getConfigSchemaFromPodFile parses a PodFile and extracts CONFIG declarations as schema.
func (b *ConfigBuilder) getConfigSchemaFromPodFile(source string) (*ConfigSchemaResponse, error) {
	prog, errs := parser.Parse(source)
	if len(errs) > 0 {
		return nil, fmt.Errorf("podfile parse errors: %v", errs)
	}

	spec := extract.Extract(prog)
	result := &ConfigSchemaResponse{
		Fields: make([]ConfigFieldResponse, 0, len(spec.Config)),
	}
	for _, cfg := range spec.Config {
		field := ConfigFieldResponse{
			Name:    cfg.Name,
			Type:    cfg.Type,
			Default: cfg.Default,
		}
		if len(cfg.Options) > 0 {
			field.Options = make([]FieldOptionResponse, 0, len(cfg.Options))
			for _, opt := range cfg.Options {
				field.Options = append(field.Options, FieldOptionResponse{Value: opt})
			}
		}
		result.Fields = append(result.Fields, field)
	}
	return result, nil
}

// buildConfigSchemaResponse converts legacy DB ConfigSchema to API response.
func (b *ConfigBuilder) buildConfigSchemaResponse(schema *agent.ConfigSchema) *ConfigSchemaResponse {
	result := &ConfigSchemaResponse{
		Fields: make([]ConfigFieldResponse, 0, len(schema.Fields)),
	}
	for _, field := range schema.Fields {
		fr := ConfigFieldResponse{
			Name:       field.Name,
			Type:       field.Type,
			Default:    field.Default,
			Required:   field.Required,
			Validation: field.Validation,
			ShowWhen:   field.ShowWhen,
		}
		if len(field.Options) > 0 {
			fr.Options = make([]FieldOptionResponse, 0, len(field.Options))
			for _, opt := range field.Options {
				fr.Options = append(fr.Options, FieldOptionResponse{Value: opt.Value})
			}
		}
		result.Fields = append(result.Fields, fr)
	}
	return result
}
