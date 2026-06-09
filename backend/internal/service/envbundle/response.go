package envbundle

import (
	"context"
	"log/slog"
	"sort"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
)

// ResponseWithValues is strict — Create uses it directly so a freshly written
// value that won't decrypt surfaces as a 500: an encryptor fault worth exposing,
// not an unreadable row persisted silently.
func (s *Service) ResponseWithValues(bundle *envbundle.EnvBundle) (*envbundle.Response, error) {
	resp := bundle.ToResponse()
	if !envbundle.IsEncryptedKind(bundle.Kind) {
		dec, err := s.decryptData(bundle.Kind, bundle.Data)
		if err != nil {
			return resp, err
		}
		if len(dec) > 0 {
			resp.ConfiguredValues = dec
		}
		return resp, nil
	}

	// Pass 1: classify by secrecy without decrypting, so a decrypt failure in
	// pass 2 still returns a response labeled with the secret field names — the
	// degraded list/get row stays informative. Sorted for a stable wire order.
	fields := make([]string, 0, len(bundle.Data))
	for k := range bundle.Data {
		if !envbundle.IsNonSecretKey(k) {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)
	if len(fields) > 0 {
		resp.ConfiguredFields = fields
	}

	// Pass 2: decryptData decrypts every key with the same all-or-nothing health
	// check the runner path (GetEffectiveForUser) applies — the list never shows
	// a bundle healthy while the runner skips it. Reuse it rather than re-walk
	// the cipher here, so the decrypt contract lives in one place.
	dec, err := s.decryptData(bundle.Kind, bundle.Data)
	if err != nil {
		return resp, err
	}
	values := make(map[string]string)
	for k, v := range dec {
		if envbundle.IsNonSecretKey(k) {
			values[k] = v
		}
	}
	if len(values) > 0 {
		resp.ConfiguredValues = values
	}
	return resp, nil
}

// ResponseWithValuesDegrading tolerates a decrypt failure instead of erroring,
// so historical corruption (e.g. a not-resubmitted secret after a key rotation)
// keeps a committed row readable, openable, and renamable rather than 500ing
// the read or update that touched it.
func (s *Service) ResponseWithValuesDegrading(ctx context.Context, bundle *envbundle.EnvBundle) *envbundle.Response {
	resp, err := s.ResponseWithValues(bundle)
	if err != nil {
		slog.WarnContext(ctx,
			"env bundle decrypt failed; serving without values",
			"bundle_id", bundle.ID,
			"name", bundle.Name,
			"kind", bundle.Kind,
			"error", err,
		)
	}
	return resp
}

// ResponsesWithValues isolates per-bundle decrypt failures via
// ResponseWithValuesDegrading — one corrupt ciphertext (e.g. after a key
// rotation) must not hide every other bundle from the settings page.
func (s *Service) ResponsesWithValues(ctx context.Context, bundles []*envbundle.EnvBundle) []*envbundle.Response {
	out := make([]*envbundle.Response, 0, len(bundles))
	for _, b := range bundles {
		out = append(out, s.ResponseWithValuesDegrading(ctx, b))
	}
	return out
}
