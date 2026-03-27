package agent

// CreateCredentialProfileParams contains parameters for creating a credential profile
type CreateCredentialProfileParams struct {
	AgentSlug  string
	Name         string
	Description  *string
	IsRunnerHost bool
	Credentials  map[string]string // Plaintext credentials to be encrypted
	IsDefault    bool
}

// UpdateCredentialProfileParams contains parameters for updating a credential profile
type UpdateCredentialProfileParams struct {
	Name         *string
	Description  *string
	IsRunnerHost *bool
	Credentials  map[string]string // If provided, will replace existing credentials
	IsDefault    *bool
	IsActive     *bool
}
