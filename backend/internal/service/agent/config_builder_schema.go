package agent

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
)

// GetConfigSchema returns the raw config schema for an agent type
// Frontend is responsible for i18n translation using: agent.{slug}.fields.{field.name}.label
func (b *ConfigBuilder) GetConfigSchema(ctx context.Context, agentTypeID int64) (*ConfigSchemaResponse, error) {
	agentType, err := b.provider.GetAgentType(ctx, agentTypeID)
	if err != nil {
		return nil, err
	}

	return b.buildConfigSchemaResponse(&agentType.ConfigSchema), nil
}

// buildConfigSchemaResponse converts internal ConfigSchema to API response
func (b *ConfigBuilder) buildConfigSchemaResponse(schema *agent.ConfigSchema) *ConfigSchemaResponse {
	result := &ConfigSchemaResponse{
		Fields: make([]ConfigFieldResponse, 0, len(schema.Fields)),
	}

	for _, field := range schema.Fields {
		fieldResponse := ConfigFieldResponse{
			Name:       field.Name,
			Type:       field.Type,
			Default:    field.Default,
			Required:   field.Required,
			Validation: field.Validation,
			ShowWhen:   field.ShowWhen,
		}

		// Convert options (without label - frontend will translate)
		if len(field.Options) > 0 {
			fieldResponse.Options = make([]FieldOptionResponse, 0, len(field.Options))
			for _, opt := range field.Options {
				fieldResponse.Options = append(fieldResponse.Options, FieldOptionResponse{
					Value: opt.Value,
				})
			}
		}

		result.Fields = append(result.Fields, fieldResponse)
	}

	return result
}
