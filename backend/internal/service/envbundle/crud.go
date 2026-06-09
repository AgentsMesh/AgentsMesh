package envbundle

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
)

// CreateParams is the input shape for creating a new bundle.
type CreateParams struct {
	OwnerScope  string
	OwnerID     int64
	AgentSlug   *string
	Name        string
	Description *string
	Kind        string
	KindPrimary bool
	Data        map[string]string
}

// UpdateParams is the input shape for updating an existing bundle. nil fields
// are skipped. Data has three states:
//   - nil           → no change
//   - &empty map    → clear all keys (caller explicitly wants the bundle empty)
//   - &non-empty    → replace stored values
type UpdateParams struct {
	Name        *string
	Description *string
	Kind        *string
	KindPrimary *bool
	Data        *map[string]string
	IsActive    *bool
}

// Create stores a new bundle. Credential-kind values are encrypted; other
// kinds are stored as plaintext. When KindPrimary is requested, the bundle
// row and the primary-promotion live in one transaction (see
// Repository.CreateWithPrimary) so we never end up with "created but failed
// to promote" partial state.
func (s *Service) Create(ctx context.Context, params *CreateParams) (*envbundle.EnvBundle, error) {
	if !validScope(params.OwnerScope) {
		return nil, ErrInvalidScope
	}
	if params.Kind == "" {
		return nil, ErrInvalidKind
	}

	exists, err := s.repo.NameExists(ctx, params.OwnerScope, params.OwnerID, params.Name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrNameExists
	}

	data, err := s.encryptData(params.Kind, params.Data)
	if err != nil {
		return nil, err
	}

	bundle := &envbundle.EnvBundle{
		OwnerScope:  params.OwnerScope,
		OwnerID:     params.OwnerID,
		AgentSlug:   params.AgentSlug,
		Name:        params.Name,
		Description: params.Description,
		Kind:        params.Kind,
		Data:        data,
		IsActive:    true,
	}

	if params.KindPrimary {
		if err := s.repo.CreateWithPrimary(ctx, bundle); err != nil {
			return nil, err
		}
		return bundle, nil
	}
	if err := s.repo.Create(ctx, bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

// Update applies the non-nil fields. Data, when non-nil and non-empty, replaces
// the stored map after re-encryption.
func (s *Service) Update(ctx context.Context, ownerScope string, ownerID, id int64, params *UpdateParams) (*envbundle.EnvBundle, error) {
	bundle, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bundle == nil || bundle.OwnerScope != ownerScope || bundle.OwnerID != ownerID {
		return nil, ErrNotFound
	}

	updates := map[string]interface{}{}
	if params.Name != nil && *params.Name != bundle.Name {
		exists, err := s.repo.NameExists(ctx, bundle.OwnerScope, bundle.OwnerID, *params.Name, &bundle.ID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrNameExists
		}
		updates["name"] = *params.Name
	}
	if params.Description != nil {
		updates["description"] = *params.Description
	}
	if params.Kind != nil && *params.Kind != bundle.Kind {
		updates["kind"] = *params.Kind
	}
	if params.IsActive != nil {
		updates["is_active"] = *params.IsActive
	}
	if params.Data != nil {
		// nil = "no change"; non-nil = "replace". An empty map explicitly
		// clears all keys. A non-empty credential update preserves untouched
		// secrets rather than wiping them — see
		// buildCredentialDataPreservingSecrets.
		effectiveKind := bundle.Kind
		if params.Kind != nil {
			effectiveKind = *params.Kind
		}
		var data envbundle.BundleData
		var err error
		// Preserving untouched secrets reuses bundle.Data's stored ciphertext,
		// which is only sound when the kind is unchanged and already encrypted.
		// On a kind switch (e.g. runtime→credential) bundle.Data holds the old
		// kind's plaintext; preserving it would store plaintext where ciphertext
		// is expected and corrupt the bundle (every later decrypt fails). A
		// switch therefore replaces rather than merges.
		preserveSecrets := envbundle.IsEncryptedKind(effectiveKind) &&
			bundle.Kind == effectiveKind &&
			len(*params.Data) > 0
		if preserveSecrets {
			data, err = s.buildCredentialDataPreservingSecrets(bundle.Data, *params.Data)
		} else {
			data, err = s.encryptData(effectiveKind, *params.Data)
		}
		if err != nil {
			return nil, err
		}
		updates["data"] = data
	}

	if len(updates) > 0 {
		if err := s.repo.Update(ctx, bundle, updates); err != nil {
			return nil, err
		}
	}

	if params.KindPrimary != nil {
		if *params.KindPrimary {
			if err := s.repo.SetPrimary(ctx, bundle); err != nil {
				return nil, err
			}
		} else if bundle.KindPrimary {
			if err := s.repo.Update(ctx, bundle, map[string]interface{}{"kind_primary": false}); err != nil {
				return nil, err
			}
		}
	}

	return s.repo.GetByID(ctx, id)
}

// Delete removes a bundle owned by (scope, ownerID).
func (s *Service) Delete(ctx context.Context, ownerScope string, ownerID, id int64) error {
	bundle, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if bundle == nil || bundle.OwnerScope != ownerScope || bundle.OwnerID != ownerID {
		return ErrNotFound
	}
	rows, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Get returns a bundle owned by (scope, ownerID).
func (s *Service) Get(ctx context.Context, ownerScope string, ownerID, id int64) (*envbundle.EnvBundle, error) {
	bundle, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bundle == nil || bundle.OwnerScope != ownerScope || bundle.OwnerID != ownerID {
		return nil, ErrNotFound
	}
	return bundle, nil
}

// SetPrimary marks a bundle as the primary in its (owner, agent_slug, kind)
// group. Clears any existing primary in that group atomically.
func (s *Service) SetPrimary(ctx context.Context, ownerScope string, ownerID, id int64) (*envbundle.EnvBundle, error) {
	bundle, err := s.Get(ctx, ownerScope, ownerID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetPrimary(ctx, bundle); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

// List returns bundles owned by (scope, ownerID), optionally filtered.
func (s *Service) List(ctx context.Context, f envbundle.OwnerFilter) ([]*envbundle.EnvBundle, error) {
	return s.repo.List(ctx, f)
}

func validScope(s string) bool {
	return s == envbundle.OwnerScopeUser || s == envbundle.OwnerScopeOrg
}
