package envbundle

// nonSecretKeys is the allowlist of credential-bundle keys whose plaintext
// value may be echoed back on the wire so the edit form can prefill them.
// These are endpoint addresses — config a user wants to see and edit, not a
// credential in their own right.
//
// CAVEAT: a base URL CAN carry an embedded secret (basic-auth userinfo or a
// token in the path/query, e.g. https://user:tok@proxy.host). We still echo
// it: prefill is the whole point of this allowlist, the value is only ever
// shown to its owner, and parsing every URL to sniff credentials is both
// unreliable (a token may hide in any component) and surprising. The edit form
// instead warns the user not to embed secrets in the URL (see the
// baseUrlSecurityHint string in the frontend credential form).
//
// Default-deny by design: a key absent here is treated as secret, so a new
// unrecognized key can never be surfaced in plaintext by mistake. Register a
// new non-secret field by adding one line. The allowlist mirrors the `TEXT`
// (vs `SECRET`) source declared per-field in the frontend credential form spec
// / AgentFile `ENV` declarations.
var nonSecretKeys = map[string]bool{
	"ANTHROPIC_BASE_URL": true,
}

// IsNonSecretKey reports whether a credential-bundle key may be echoed back in
// plaintext. Callers must treat a false result as "secret": return the key
// name only (never the value) on read, and preserve the stored value on write
// when the field is left blank.
func IsNonSecretKey(key string) bool {
	return nonSecretKeys[key]
}
