package envbundle

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ResponseWithValues_Credential_SplitsByKey(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "tok-secret",
			"ANTHROPIC_BASE_URL":   "https://api.example.com",
		},
	})

	resp, err := svc.ResponseWithValues(b)
	require.NoError(t, err)

	// Secret key → name only, never the value. ElementsMatch (not Equal) because
	// ConfiguredFields order is not guaranteed for multiple secrets.
	assert.ElementsMatch(t, []string{"ANTHROPIC_AUTH_TOKEN"}, resp.ConfiguredFields)
	assert.NotContains(t, resp.ConfiguredValues, "ANTHROPIC_AUTH_TOKEN")
	// Non-secret key → plaintext echoed back for edit-form prefill.
	assert.Equal(t, "https://api.example.com", resp.ConfiguredValues["ANTHROPIC_BASE_URL"])
	assert.NotContains(t, resp.ConfiguredFields, "ANTHROPIC_BASE_URL")
}

func TestService_ResponsesWithValues_IsolatesDecryptFailure(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// Healthy credential bundle: its non-secret base URL decrypts and round-trips.
	_, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "good", Kind: envbundle.KindCredential,
		Data: map[string]string{"ANTHROPIC_BASE_URL": "https://api.example.com"},
	})
	require.NoError(t, err)

	// Corrupt bundle: invalid ciphertext inserted directly to bypass encryption,
	// simulating a key rotation. One bad bundle must not 500 the whole list.
	require.NoError(t, db.Exec(
		`INSERT INTO env_bundles (owner_scope, owner_id, name, kind, kind_primary, data, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		envbundle.OwnerScopeUser, 1, "corrupt",
		envbundle.KindCredential, false,
		`{"ANTHROPIC_AUTH_TOKEN":"garbage-cipher","ANTHROPIC_BASE_URL":"not-ciphertext"}`, true,
	).Error)

	bundles, err := svc.List(ctx, envbundle.OwnerFilter{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
	})
	require.NoError(t, err)

	responses := svc.ResponsesWithValues(ctx, bundles)
	require.Len(t, responses, 2, "both bundles listed; the corrupt one is not dropped")

	byName := make(map[string]*envbundle.Response, len(responses))
	for _, r := range responses {
		byName[r.Name] = r
	}
	assert.Equal(t, "https://api.example.com",
		byName["good"].ConfiguredValues["ANTHROPIC_BASE_URL"])
	// Corrupt bundle still appears (visible to repair/delete), degraded to no
	// values. Secret field names need no decryption, so they survive.
	require.Contains(t, byName, "corrupt")
	assert.Empty(t, byName["corrupt"].ConfiguredValues)
	assert.ElementsMatch(t, []string{"ANTHROPIC_AUTH_TOKEN"}, byName["corrupt"].ConfiguredFields)
}

func TestService_ResponsesWithValues_CorruptSecretDegradesLikeRunner(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// Healthy bundle with a valid base URL AND a valid token, both encrypted.
	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{
			"ANTHROPIC_BASE_URL":   "https://api.example.com",
			"ANTHROPIC_AUTH_TOKEN": "tok-secret",
		},
	})
	require.NoError(t, err)

	// Corrupt ONLY the secret's ciphertext, leaving the base URL valid. The list
	// must still degrade this bundle (no values) because the runner decrypts
	// every key and would skip it — the two health checks must agree, else the
	// user sees a "healthy" bundle the pod silently never receives.
	got, err := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	require.NoError(t, err)
	corrupt := envbundle.BundleData{}
	for k, v := range got.Data {
		corrupt[k] = v
	}
	corrupt["ANTHROPIC_AUTH_TOKEN"] = "garbage"
	require.NoError(t, db.Model(&envbundle.EnvBundle{}).
		Where("id = ?", b.ID).Update("data", corrupt).Error)

	bundles, err := svc.List(ctx, envbundle.OwnerFilter{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
	})
	require.NoError(t, err)

	responses := svc.ResponsesWithValues(ctx, bundles)
	require.Len(t, responses, 1)
	assert.Empty(t, responses[0].ConfiguredValues,
		"a corrupt secret degrades the whole bundle in the list, matching the runner skip")
	assert.ElementsMatch(t, []string{"ANTHROPIC_AUTH_TOKEN"}, responses[0].ConfiguredFields)
}

func TestService_Update_Credential_ResubmittedSecretOverwrites(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{"ANTHROPIC_AUTH_TOKEN": "old-tok"},
	})

	newData := map[string]string{"ANTHROPIC_AUTH_TOKEN": "new-tok"}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.Equal(t, "new-tok", dec["ANTHROPIC_AUTH_TOKEN"],
		"a resubmitted secret overwrites the old ciphertext, not preserved as stale")
}

func TestService_Update_Credential_XorSwitchDropsOldSibling(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Stored with an Auth Token (XOR auth method).
	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{"ANTHROPIC_AUTH_TOKEN": "tok"},
	})

	// User switches to API Key: the form submits the new key plus the deselected
	// sibling as an empty value. The old token must be dropped, not preserved —
	// else the pod receives both ANTHROPIC_API_KEY and ANTHROPIC_AUTH_TOKEN.
	newData := map[string]string{
		"ANTHROPIC_API_KEY":    "sk-key",
		"ANTHROPIC_AUTH_TOKEN": "",
	}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.Equal(t, "sk-key", dec["ANTHROPIC_API_KEY"])
	assert.NotContains(t, dec, "ANTHROPIC_AUTH_TOKEN",
		"a deselected XOR sibling submitted empty is dropped, not merge-preserved")
}

func TestService_Create_Credential_DropsEmptyValues(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// The form sends deselected XOR siblings as empty strings. Create must drop
	// them — storing an encrypted "" would surface as a bogus "configured" key
	// and inject a blank env var into the pod alongside the real one.
	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{
			"ANTHROPIC_API_KEY":    "sk-key",
			"ANTHROPIC_AUTH_TOKEN": "",
		},
	})
	require.NoError(t, err)

	var raw envbundle.EnvBundle
	require.NoError(t, db.First(&raw, b.ID).Error)
	assert.NotContains(t, raw.Data, "ANTHROPIC_AUTH_TOKEN",
		"empty credential value must not be stored")
	assert.Contains(t, raw.Data, "ANTHROPIC_API_KEY")
}

func TestService_Update_Credential_DeletesStandaloneSecretViaEmptyValue(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// loopal-style bundle: two independent (non-oneof) secret keys. The form's
	// explicit "remove" action submits the deleted key as an empty string while
	// leaving the untouched key out of the payload entirely (blank = keep). The
	// empty one must be dropped; the omitted one preserved. Before the per-key
	// merge this was the reverse of the bug the merge introduced — a standalone
	// secret could not be deleted at all once "leave blank = keep" was in force.
	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{
			"ANTHROPIC_API_KEY": "keep-me",
			"OPENAI_API_KEY":    "remove-me",
		},
	})

	newData := map[string]string{"OPENAI_API_KEY": ""}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.NotContains(t, dec, "OPENAI_API_KEY",
		"a standalone secret submitted empty is deleted, not preserved")
	assert.Equal(t, "keep-me", dec["ANTHROPIC_API_KEY"],
		"an omitted (untouched) secret is preserved")
}

func TestService_ResponseWithValuesDegrading_CorruptReturnsNoError(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{"ANTHROPIC_AUTH_TOKEN": "tok"},
	})
	require.NoError(t, err)

	// Corrupt the stored ciphertext (simulating a key rotation). The degrading
	// read — now used by Update's response too, not just Get — must return no
	// error so editing the bundle (e.g. its name) can't 500 on historical
	// corruption the write never touched.
	require.NoError(t, db.Model(&envbundle.EnvBundle{}).
		Where("id = ?", b.ID).
		Update("data", envbundle.BundleData{"ANTHROPIC_AUTH_TOKEN": "garbage"}).Error)

	reloaded, err := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	require.NoError(t, err)

	resp := svc.ResponseWithValuesDegrading(ctx, reloaded)
	require.NotNil(t, resp)
	assert.ElementsMatch(t, []string{"ANTHROPIC_AUTH_TOKEN"}, resp.ConfiguredFields,
		"secret field names survive without decryption")
	assert.Empty(t, resp.ConfiguredValues, "no values when the bundle won't decrypt")
}

func TestService_ResponseWithValues_Corrupt_ReturnsError(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{"ANTHROPIC_AUTH_TOKEN": "tok"},
	})
	require.NoError(t, err)

	// Corrupt the ciphertext. The strict ResponseWithValues — used by Create —
	// must surface the failure as an error so CreateEnvBundle 500s. A freshly
	// written value that won't decrypt is an encryptor fault to expose, not the
	// historical corruption the degrading read tolerates.
	require.NoError(t, db.Model(&envbundle.EnvBundle{}).
		Where("id = ?", b.ID).
		Update("data", envbundle.BundleData{"ANTHROPIC_AUTH_TOKEN": "garbage"}).Error)

	reloaded, err := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	require.NoError(t, err)

	_, err = svc.ResponseWithValues(reloaded)
	require.ErrorContains(t, err, "decrypt",
		"strict path surfaces the decrypt failure specifically (so Create 500s)")
}
