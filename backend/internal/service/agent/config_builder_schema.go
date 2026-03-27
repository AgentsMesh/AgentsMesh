package agent

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/podfile/extract"
	"github.com/anthropics/agentsmesh/podfile/parser"
)

// GetConfigSchema returns the config schema for an agent.
// CONFIG declarations are extracted from the PodFile source.
func (b *ConfigBuilder) GetConfigSchema(ctx context.Context, agentSlug string) (*ConfigSchemaResponse, error) {
	agentDef, err := b.provider.GetAgent(ctx, agentSlug)
	if err != nil {
		return nil, err
	}
	if agentDef.PodfileSource != nil && *agentDef.PodfileSource != "" {
		return b.getConfigSchemaFromPodFile(*agentDef.PodfileSource)
	}
	// No PodFile = empty schema
	return &ConfigSchemaResponse{Fields: []ConfigFieldResponse{}}, nil
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
