package apikey

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

func (k *APIKey) ValidateIdentifiers() error {
	if k.Slug == nil {
		return nil
	}
	return slugkit.ValidateIdentifier("api_keys.slug", *k.Slug)
}
