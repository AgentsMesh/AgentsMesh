package envbundle

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testEncryptKey = "test-encrypt-key"

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewEnvBundleRepository(db)
	return NewService(repo, crypto.NewEncryptor(testEncryptKey)), db
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// ---------- Create ----------

func TestService_Create_CredentialKindEncrypts(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser,
		OwnerID:    1,
		AgentSlug:  strPtr("claude-code"),
		Name:       "work-creds",
		Kind:       envbundle.KindCredential,
		Data:       map[string]string{"ANTHROPIC_API_KEY": "sk-secret"},
	})
	require.NoError(t, err)
	require.NotZero(t, b.ID)
	assert.False(t, b.KindPrimary)

	// Stored ciphertext != plaintext.
	var raw envbundle.EnvBundle
	require.NoError(t, db.First(&raw, b.ID).Error)
	assert.NotEqual(t, "sk-secret", raw.Data["ANTHROPIC_API_KEY"],
		"credential value should be encrypted at rest")
}

func TestService_Create_RuntimeKindStoresPlaintext(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	b, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser,
		OwnerID:    1,
		Name:       "log-prefs",
		Kind:       envbundle.KindRuntime,
		Data:       map[string]string{"LOG_LEVEL": "debug"},
	})
	require.NoError(t, err)

	var raw envbundle.EnvBundle
	require.NoError(t, db.First(&raw, b.ID).Error)
	assert.Equal(t, "debug", raw.Data["LOG_LEVEL"],
		"non-encrypted kinds round-trip plaintext")
}

func TestService_Create_WithPrimary_DemotesExisting(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	first, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "first",
		Kind: envbundle.KindCredential, KindPrimary: true,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)
	assert.True(t, first.KindPrimary)

	// Creating a second primary in the same (agent_slug, kind) group must
	// atomically demote the first — CreateWithPrimary uses one transaction.
	second, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "second",
		Kind: envbundle.KindCredential, KindPrimary: true,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)
	assert.True(t, second.KindPrimary)

	got, err := svc.Get(ctx, envbundle.OwnerScopeUser, 1, first.ID)
	require.NoError(t, err)
	assert.False(t, got.KindPrimary, "first should have been demoted")
}

func TestService_Create_NameCollision(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "dup", Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)

	_, err = svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "dup", Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v2"},
	})
	assert.ErrorIs(t, err, ErrNameExists)
}

func TestService_Create_InvalidScopeAndKind(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateParams{
		OwnerScope: "garbage", OwnerID: 1, Name: "x",
		Kind: envbundle.KindCredential,
	})
	assert.ErrorIs(t, err, ErrInvalidScope)

	_, err = svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1, Name: "x", Kind: "",
	})
	assert.ErrorIs(t, err, ErrInvalidKind)
}

// ---------- Update (Data tri-state) ----------

func TestService_Update_NilData_LeavesUnchanged(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "n", Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})

	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Description: strPtr("new desc"),
		Data:        nil, // explicit: don't touch
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.Equal(t, "v", dec["K"], "Data=nil must not clear keys")
}

func TestService_Update_EmptyData_ClearsKeys(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "n", Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})

	empty := map[string]string{}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &empty,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	assert.Empty(t, got.Data, "&empty map clears the stored values")
}

func TestService_Update_NonEmptyData_Replaces(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Runtime (non-encrypted) kind keeps the plain replace contract: non-empty
	// Data wholesale replaces stored keys. Credential merge semantics (preserve
	// untouched secrets) are covered separately below.
	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "n", Kind: envbundle.KindRuntime,
		Data: map[string]string{"OLD_KEY": "v"},
	})

	newData := map[string]string{"NEW_KEY": "x"}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.NotContains(t, dec, "OLD_KEY")
	assert.Equal(t, "x", dec["NEW_KEY"])
}

func TestService_Update_Credential_PreservesUnsubmittedSecrets(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// Editing a credential profile: the auth token is left blank to keep its
	// current value, only the non-secret base URL is resubmitted. The token
	// must survive — wiping it is the bug this guards.
	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "creds", Kind: envbundle.KindCredential,
		Data: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "tok-secret",
			"ANTHROPIC_BASE_URL":   "https://old.example.com",
		},
	})

	newData := map[string]string{"ANTHROPIC_BASE_URL": "https://new.example.com"}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	dec, _ := svc.decryptData(got.Kind, got.Data)
	assert.Equal(t, "tok-secret", dec["ANTHROPIC_AUTH_TOKEN"],
		"unsubmitted secret must be preserved, not wiped")
	assert.Equal(t, "https://new.example.com", dec["ANTHROPIC_BASE_URL"],
		"resubmitted non-secret takes the new value")
}

func TestService_Update_KindSwitch_ReplacesInsteadOfPreserving(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	// A runtime bundle stores plaintext. Switching it to credential while
	// resubmitting only one key must NOT carry the old plaintext keys over as
	// if they were ciphertext — that would corrupt the bundle so every later
	// decrypt fails. A kind switch replaces; only a same-kind edit merges.
	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "switch", Kind: envbundle.KindRuntime,
		Data: map[string]string{"OLD_PLAINTEXT": "bar"},
	})

	newData := map[string]string{"ANTHROPIC_API_KEY": "sk-new"}
	_, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		Kind: strPtr(envbundle.KindCredential),
		Data: &newData,
	})
	require.NoError(t, err)

	got, _ := svc.Get(ctx, envbundle.OwnerScopeUser, 1, b.ID)
	require.Equal(t, envbundle.KindCredential, got.Kind)

	dec, err := svc.decryptData(got.Kind, got.Data)
	require.NoError(t, err, "bundle must decrypt cleanly after a kind switch")
	assert.NotContains(t, dec, "OLD_PLAINTEXT",
		"old plaintext key must not survive as bogus ciphertext")
	assert.Equal(t, "sk-new", dec["ANTHROPIC_API_KEY"])
}

func TestService_Update_KindPrimary_TogglesPrimary(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	b, _ := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "n",
		Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})

	got, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		KindPrimary: boolPtr(true),
	})
	require.NoError(t, err)
	assert.True(t, got.KindPrimary)

	got2, err := svc.Update(ctx, envbundle.OwnerScopeUser, 1, b.ID, &UpdateParams{
		KindPrimary: boolPtr(false),
	})
	require.NoError(t, err)
	assert.False(t, got2.KindPrimary)
}

// ---------- decrypt strict failure ----------

func TestDecryptData_StrictFail_PropagatesError(t *testing.T) {
	svc, _ := newTestService(t)

	corrupt := envbundle.BundleData{
		"K1": "this-is-not-valid-ciphertext",
	}
	_, err := svc.decryptData(envbundle.KindCredential, corrupt)
	require.Error(t, err, "credential kind must NEVER silently treat ciphertext-as-plaintext")
}

func TestDecryptData_NonEncryptedKind_RoundtripsPlaintext(t *testing.T) {
	svc, _ := newTestService(t)

	plain := envbundle.BundleData{"LOG_LEVEL": "info"}
	out, err := svc.decryptData(envbundle.KindRuntime, plain)
	require.NoError(t, err)
	assert.Equal(t, "info", out["LOG_LEVEL"])
}

// ---------- GetEffectiveForUser ----------

func TestGetEffectiveForUser_FiltersInactiveAndByAgent(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// Active credential for claude-code
	_, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "active-cc",
		Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)

	// Universal (agent_slug=NULL) runtime — should match any agent
	_, err = svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		Name: "universal", Kind: envbundle.KindRuntime,
		Data: map[string]string{"LOG_LEVEL": "info"},
	})
	require.NoError(t, err)

	// Inactive credential — should be skipped
	inactive, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "inactive",
		Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(inactive).Update("is_active", false).Error)

	// Bundle for a different agent — should NOT appear in claude-code list
	_, err = svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("codex-cli"), Name: "other-agent",
		Kind: envbundle.KindCredential,
		Data: map[string]string{"K": "v"},
	})
	require.NoError(t, err)

	got, err := svc.GetEffectiveForUser(ctx, 1, 0, "claude-code")
	require.NoError(t, err)

	names := make(map[string]bool, len(got))
	for _, b := range got {
		names[b.Name] = true
	}
	assert.True(t, names["active-cc"], "active claude-code bundle is included")
	assert.True(t, names["universal"], "NULL agent_slug bundle is universal")
	assert.False(t, names["inactive"], "inactive bundle is skipped")
	assert.False(t, names["other-agent"], "different agent bundle is excluded")
}

func TestGetEffectiveForUser_SkipsBundlesThatFailToDecrypt(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	// Healthy bundle
	healthy, err := svc.Create(ctx, &CreateParams{
		OwnerScope: envbundle.OwnerScopeUser, OwnerID: 1,
		AgentSlug: strPtr("claude-code"), Name: "good",
		Kind: envbundle.KindCredential,
		Data: map[string]string{"GOOD_KEY": "g"},
	})
	require.NoError(t, err)
	_ = healthy

	// Manually corrupt a credential bundle by writing invalid ciphertext
	// directly to the DB — simulates a key rotation / data corruption.
	require.NoError(t, db.Exec(
		`INSERT INTO env_bundles (owner_scope, owner_id, agent_slug, name, kind, kind_primary, data, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		envbundle.OwnerScopeUser, 1, "claude-code", "corrupt",
		envbundle.KindCredential, false, `{"BAD":"not-actually-ciphertext"}`, true,
	).Error)

	got, err := svc.GetEffectiveForUser(ctx, 1, 0, "claude-code")
	require.NoError(t, err, "the request must not fail even when one bundle is corrupt")

	names := make(map[string]bool, len(got))
	for _, b := range got {
		names[b.Name] = true
	}
	assert.True(t, names["good"], "healthy bundle still loads")
	assert.False(t, names["corrupt"], "decrypt-failing bundle is skipped (logged at ERROR by the service)")
}
