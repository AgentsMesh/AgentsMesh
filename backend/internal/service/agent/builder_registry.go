package agent

import (
	"sync"
)

// AgentBuilderRegistry manages agent builder strategies.
// It provides lookup by agent slug and falls back to BaseAgentBuilder
// for unknown agent types.
type AgentBuilderRegistry struct {
	mu       sync.RWMutex
	builders map[string]AgentBuilder
	fallback AgentBuilder
}

// NewAgentBuilderRegistry creates a new registry with default builders registered.
func NewAgentBuilderRegistry() *AgentBuilderRegistry {
	r := &AgentBuilderRegistry{
		builders: make(map[string]AgentBuilder),
		fallback: NewBaseAgentBuilder("default"),
	}

	// Register built-in agent builders
	r.Register(NewClaudeCodeBuilder())
	r.Register(NewCodexCLIBuilder())
	r.Register(NewGeminiCLIBuilder())
	r.Register(NewAiderBuilder())
	r.Register(NewOpenCodeBuilder())

	return r
}

// Register adds a builder to the registry.
// The builder's Slug() is used as the key.
func (r *AgentBuilderRegistry) Register(builder AgentBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[builder.Slug()] = builder
}

// Get returns the builder for the given slug.
// Returns the fallback builder if no specific builder is registered.
func (r *AgentBuilderRegistry) Get(slug string) AgentBuilder {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if builder, ok := r.builders[slug]; ok {
		return builder
	}
	return r.fallback
}

// Has checks if a builder is registered for the given slug.
func (r *AgentBuilderRegistry) Has(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.builders[slug]
	return ok
}

// List returns all registered builder slugs.
func (r *AgentBuilderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	slugs := make([]string, 0, len(r.builders))
	for slug := range r.builders {
		slugs = append(slugs, slug)
	}
	return slugs
}

// SetFallback sets the fallback builder used for unknown agent types.
func (r *AgentBuilderRegistry) SetFallback(builder AgentBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fallback = builder
}
