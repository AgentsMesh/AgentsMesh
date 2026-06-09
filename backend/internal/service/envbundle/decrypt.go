package envbundle

import (
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
)

// encryptData encrypts every value when the kind demands it; for non-encrypted
// kinds the map is returned as-is (still backed by JSONB at the DB layer).
func (s *Service) encryptData(kind string, data map[string]string) (envbundle.BundleData, error) {
	if !envbundle.IsEncryptedKind(kind) {
		out := make(envbundle.BundleData, len(data))
		for k, v := range data {
			out[k] = v
		}
		return out, nil
	}
	out := make(envbundle.BundleData, len(data))
	for k, v := range data {
		if v == "" {
			// An empty credential value means "no key": never store a blank
			// secret/URL. This is how a deselected XOR sibling or a cleared
			// field is dropped — uniformly on create, update, and kind-switch.
			continue
		}
		enc, err := s.encryptor.Encrypt(v)
		if err != nil {
			return nil, err
		}
		out[k] = enc
	}
	return out, nil
}

// buildCredentialDataPreservingSecrets merges a credential-bundle update so a
// blank secret field keeps its stored value instead of being wiped. Newly
// submitted keys are (re-)encrypted; secret keys (per IsNonSecretKey) absent
// from the submission keep their old ciphertext — "leave blank to keep
// current" is the contract the edit form's secret inputs promise. Non-secret
// keys (e.g. base URL) follow the submission verbatim: absent means cleared.
func (s *Service) buildCredentialDataPreservingSecrets(
	old envbundle.BundleData, newPlain map[string]string,
) (envbundle.BundleData, error) {
	// encryptData skips empty values, so a submitted blank (a deselected XOR
	// sibling or a cleared field) is dropped rather than stored.
	out, err := s.encryptData(envbundle.KindCredential, newPlain)
	if err != nil {
		return nil, err
	}
	for k, cipher := range old {
		if _, present := newPlain[k]; present {
			continue // submitted: encrypted above if non-empty, dropped if empty
		}
		if envbundle.IsNonSecretKey(k) {
			continue // non-secret absent → cleared
		}
		out[k] = cipher // secret absent → preserved ("leave blank to keep")
	}
	return out, nil
}

// decryptData reverses encryptData. Returns plaintext KV regardless of kind.
// For encrypted kinds, ANY value that fails to decrypt aborts the whole bundle —
// the previous "silently treat ciphertext as plaintext" fallback would leak
// ciphertext into Pod env, which is a security and observability hazard.
// Bundles with corrupt rows must be rotated by the operator, not transparently
// degraded by this layer.
func (s *Service) decryptData(kind string, data envbundle.BundleData) (map[string]string, error) {
	out := make(map[string]string, len(data))
	if !envbundle.IsEncryptedKind(kind) {
		for k, v := range data {
			out[k] = v
		}
		return out, nil
	}
	for k, v := range data {
		dec, err := s.encryptor.Decrypt(v)
		if err != nil {
			return nil, fmt.Errorf("envbundle: failed to decrypt key %q: %w", k, err)
		}
		out[k] = dec
	}
	return out, nil
}
