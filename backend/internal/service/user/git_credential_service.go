package user

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

func (s *Service) CreateGitCredential(ctx context.Context, userID int64, req *CreateGitCredentialRequest) (*user.GitCredential, error) {
	if !user.IsValidCredentialType(req.CredentialType) {
		return nil, ErrInvalidCredentialType
	}

	exists, err := s.repo.GitCredentialNameExists(ctx, userID, req.Name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCredentialAlreadyExists
	}

	credential := &user.GitCredential{
		UserID:         userID,
		Name:           req.Name,
		CredentialType: req.CredentialType,
		IsDefault:      false,
	}

	if err := s.processCredentialType(ctx, userID, credential, req); err != nil {
		return nil, err
	}

	if req.HostPattern != "" {
		credential.HostPattern = &req.HostPattern
	}

	if err := s.repo.CreateGitCredential(ctx, credential); err != nil {
		slog.ErrorContext(ctx, "failed to create git credential",
			"user_id", userID, "name", req.Name, "type", req.CredentialType, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "git credential created",
		"user_id", userID, "credential_id", credential.ID, "type", req.CredentialType)
	return credential, nil
}

func (s *Service) processCredentialType(ctx context.Context, userID int64, credential *user.GitCredential, req *CreateGitCredentialRequest) error {
	switch req.CredentialType {
	case user.CredentialTypeRunnerLocal:
		return nil

	case user.CredentialTypeOAuth:
		if req.RepositoryProviderID == nil {
			return ErrProviderIDRequired
		}
		_, err := s.GetRepositoryProvider(ctx, userID, *req.RepositoryProviderID)
		if err != nil {
			return err
		}
		credential.RepositoryProviderID = req.RepositoryProviderID

	case user.CredentialTypePAT:
		if req.PAT == "" {
			return errors.New("PAT is required for pat type")
		}
		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(req.PAT, s.encryptionKey)
			if err != nil {
				return err
			}
			credential.PATEncrypted = &encrypted
		} else {
			credential.PATEncrypted = &req.PAT
		}

	case user.CredentialTypeSSHKey:
		if req.PrivateKey == "" {
			return errors.New("private key is required for ssh_key type")
		}

		privateKey, publicKey, fingerprint, err := parseSSHKey(req.PrivateKey, req.PublicKey)
		if err != nil {
			return err
		}

		credential.PublicKey = &publicKey
		credential.Fingerprint = &fingerprint

		if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
			if err != nil {
				return err
			}
			credential.PrivateKeyEncrypted = &encrypted
		} else {
			credential.PrivateKeyEncrypted = &privateKey
		}
	}

	return nil
}

func (s *Service) GetGitCredential(ctx context.Context, userID, credentialID int64) (*user.GitCredential, error) {
	credential, err := s.repo.GetGitCredentialWithProvider(ctx, userID, credentialID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	return credential, nil
}

func (s *Service) ListGitCredentials(ctx context.Context, userID int64) ([]*user.GitCredential, error) {
	return s.repo.ListGitCredentialsWithProvider(ctx, userID)
}

func (s *Service) UpdateGitCredential(ctx context.Context, userID, credentialID int64, req *UpdateGitCredentialRequest) (*user.GitCredential, error) {
	credential, err := s.GetGitCredential(ctx, userID, credentialID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil && *req.Name != "" {
		exists, err := s.repo.GitCredentialNameExists(ctx, userID, *req.Name, &credentialID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCredentialAlreadyExists
		}
		updates["name"] = *req.Name
	}

	if req.HostPattern != nil {
		if *req.HostPattern == "" {
			updates["host_pattern"] = nil
		} else {
			updates["host_pattern"] = *req.HostPattern
		}
	}

	if err := s.applyCredentialTypeUpdates(credential, req, updates); err != nil {
		return nil, err
	}

	if len(updates) == 0 {
		return credential, nil
	}

	if err := s.repo.UpdateGitCredential(ctx, credential, updates); err != nil {
		slog.ErrorContext(ctx, "failed to update git credential",
			"user_id", userID, "credential_id", credentialID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "git credential updated", "user_id", userID, "credential_id", credentialID)
	return s.GetGitCredential(ctx, userID, credentialID)
}

func (s *Service) applyCredentialTypeUpdates(credential *user.GitCredential, req *UpdateGitCredentialRequest, updates map[string]interface{}) error {
	if req.PAT != nil && credential.CredentialType == user.CredentialTypePAT {
		if *req.PAT == "" {
			updates["pat_encrypted"] = nil
		} else if s.encryptionKey != "" {
			encrypted, err := crypto.EncryptWithKey(*req.PAT, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["pat_encrypted"] = encrypted
		} else {
			updates["pat_encrypted"] = *req.PAT
		}
	}

	if req.PrivateKey != nil && credential.CredentialType == user.CredentialTypeSSHKey {
		if *req.PrivateKey == "" {
			updates["private_key_encrypted"] = nil
			updates["public_key"] = nil
			updates["fingerprint"] = nil
		} else {
			privateKey, publicKey, fingerprint, err := parseSSHKey(*req.PrivateKey, "")
			if err != nil {
				return err
			}

			updates["public_key"] = publicKey
			updates["fingerprint"] = fingerprint

			if s.encryptionKey != "" {
				encrypted, err := crypto.EncryptWithKey(privateKey, s.encryptionKey)
				if err != nil {
					return err
				}
				updates["private_key_encrypted"] = encrypted
			} else {
				updates["private_key_encrypted"] = privateKey
			}
		}
	}

	return nil
}

func (s *Service) DeleteGitCredential(ctx context.Context, userID, credentialID int64) error {
	credential, err := s.repo.GetGitCredentialWithProvider(ctx, userID, credentialID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return ErrCredentialNotFound
		}
		return err
	}

	if credential.IsDefault {
		if err := s.repo.ClearUserDefaultCredential(ctx, userID, credentialID); err != nil {
			return err
		}
	}

	rowsAffected, err := s.repo.DeleteGitCredential(ctx, userID, credentialID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete git credential",
			"user_id", userID, "credential_id", credentialID, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrCredentialNotFound
	}
	slog.InfoContext(ctx, "git credential deleted", "user_id", userID, "credential_id", credentialID)
	return nil
}
