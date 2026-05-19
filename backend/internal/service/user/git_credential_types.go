package user

import (
	"errors"
)

var (
	ErrCredentialNotFound      = errors.New("git credential not found")
	ErrCredentialAlreadyExists = errors.New("git credential already exists with this name")
	ErrInvalidCredentialType   = errors.New("invalid credential type")
	ErrInvalidSSHKey           = errors.New("invalid SSH key format")
	ErrProviderIDRequired      = errors.New("repository_provider_id is required for oauth type")
)

type CreateGitCredentialRequest struct {
	Name                 string
	CredentialType       string // runner_local, oauth, pat, ssh_key
	RepositoryProviderID *int64 // Required for oauth type
	PAT                  string // For pat type
	PublicKey            string // For ssh_key type (can be generated)
	PrivateKey           string // For ssh_key type
	HostPattern          string // Optional host pattern
}

type UpdateGitCredentialRequest struct {
	Name        *string
	PAT         *string // For pat type
	PrivateKey  *string // For ssh_key type
	HostPattern *string
}

type DecryptedCredential struct {
	Type          string // runner_local, oauth, pat, ssh_key
	Token         string // For oauth and pat types
	SSHPrivateKey string // For ssh_key type
	SSHPublicKey  string // For ssh_key type
}
